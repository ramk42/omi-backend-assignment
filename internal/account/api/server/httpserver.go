package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/ramk42/omi-backend-assignment/internal/account/usecase"
	"github.com/ramk42/omi-backend-assignment/pkg/logsreporting"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Run(ctx context.Context, logsReportingProducer logsreporting.Producer) {

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(ctx)
	// The HTTP Server
	server := &http.Server{Addr: "0.0.0.0:8080", Handler: service(serverCtx, logsReportingProducer)}

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig
		log.Info().Msg("shutdown signal received - Starting graceful shutdown...")
		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, _ := context.WithTimeout(serverCtx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
				log.Fatal().Err(shutdownCtx.Err()).Msg("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Trigger graceful shutdown
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal().Msg("failed to shutdown server")
			os.Exit(1)
		}
		serverStopCtx()
	}()

	// Run the server
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Msg("failed to start server")
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
}

func service(serverCtx context.Context, logsReportingProducer logsreporting.Producer) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(RequestLogger)
	r.Use(middleware.RealIP)

	r.Route("/accounts", func(r chi.Router) {
		r.Use(AuditLogMiddleware(serverCtx, logsReportingProducer, "account"))
		accounUsecase := usecase.NewAccount()
		accountHandler := &AccountHandler{accountUsecase: accounUsecase}
		r.Patch("/{resourceID}", accountHandler.Patch)
	})

	return r
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.statusCode = statusCode
	rr.ResponseWriter.WriteHeader(statusCode)
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		requestID := middleware.GetReqID(r.Context())
		log.Info().
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("ip", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).Msg("Request received")

		logger := log.With().Str("request_id", requestID).Logger()
		ctx := logger.WithContext(r.Context())
		r = r.WithContext(ctx)

		next.ServeHTTP(recorder, r)

		duration := time.Since(start)
		logger.Info().
			Int("status", recorder.statusCode).
			Dur("duration", duration).
			Msg("request completed")
	})
}

func AuditLogMiddleware(serverCtx context.Context, logsReportingProducer logsreporting.Producer, resourceType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := middleware.GetReqID(r.Context())
			var requestPayload json.RawMessage
			var compactPayload bytes.Buffer
			if r.Body != nil {
				bodyBytes, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(bodyBytes, &requestPayload)
				_ = json.Compact(&compactPayload, bodyBytes)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
			recorder := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK, body: new(bytes.Buffer)}
			next.ServeHTTP(recorder, r)

			var responsePayload json.RawMessage
			if recorder.body.Len() > 0 {
				_ = json.Unmarshal(recorder.body.Bytes(), &responsePayload)
			}

			auditLog := &logsreporting.AuditLog{} // we can optimize this using sync.Pool

			subject := "event:" + resourceType
			if resourceType != "" {
				subject = subject + ":" + chi.URLParam(r, "resourceID")
			}

			resourceID := chi.URLParam(r, "resourceID")
			auditLog.SpecVersion = "1.0"
			auditLog.ID = uuid.NewString()
			auditLog.Source = "backend.api" // to in consts
			auditLog.Type = "audit.event"   // to in consts etc...
			auditLog.Subject = subject
			auditLog.Timestamp = time.Now().UTC()
			auditLog.Actor = map[string]string{
				"id": uuid.NewString(), // we assume that we have authenticated user
			}
			auditLog.Action = r.Method
			auditLog.Resource = map[string]any{
				"id":   resourceID,
				"type": resourceType,
				"attributes": map[string]string{
					"id":         resourceID,
					"type":       resourceType,
					"attributes": compactPayload.String(),
				},
			}
			auditLog.Metadata = map[string]string{
				"request_id":      requestID,
				"response_status": strconv.Itoa(recorder.statusCode),
				"protocol":        r.Proto,
			}
			// we don't block the request to publish the audit event
			go func() {
				err := logsReportingProducer.Publish(serverCtx, auditLog)
				if err != nil {
					log.Error().Err(err).Msg("failure to publish audit event")
					return
				}
			}()
		})
	}
}
