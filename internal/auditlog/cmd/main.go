package main

import (
	"context"
	"errors"
	"github.com/ramk42/omi-backend-assignment/internal/auditlog/consumer"
	"github.com/ramk42/omi-backend-assignment/internal/auditlog/repository"
	"github.com/ramk42/omi-backend-assignment/internal/auditlog/usecase"
	"github.com/ramk42/omi-backend-assignment/pkg/database"
	"github.com/ramk42/omi-backend-assignment/pkg/env"
	"github.com/ramk42/omi-backend-assignment/pkg/natsclient"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	ctx := context.Background()
	ctx, consumersStopCtx := context.WithCancel(ctx)

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig
		log.Info().Msg("shutdown signal received - Starting graceful shutdown...")
		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, _ := context.WithTimeout(ctx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
				log.Fatal().Err(shutdownCtx.Err()).Msg("shutdown timed out")
			}
		}()

		consumersStopCtx()
	}()

	log.Info().Msg("Starting audit log consumer...")
	db := database.New(ctx, env.MustGetEnv[string]("DATABASE_URL"))
	audiLogRepo := repository.NewAuditEventRepository(db)

	auditLogUsecase := usecase.NewAuditLog(
		ctx,
		audiLogRepo,
		env.MustGetEnv[int]("AUDIT_CONSUMTION_BATCH_SIZE"),
		time.Duration(env.MustGetEnv[int]("AUDIT_CONSUMTION_BATCH_FLUSH_INTERVAL_SEC"))*time.Second,
	)
	natsURL := env.MustGetEnv[string]("NATS_URL")
	conn, err := natsclient.New(natsURL)
	if err != nil {
		log.Fatal().Err(err).Str("nats_url", natsURL).Msg("failed to connect to nats")
	}
	auditLogConsumer := consumer.NewAuditLog(auditLogUsecase, conn)
	err = auditLogConsumer.Start(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start audit log consumer")
		return
	}
	log.Info().Msg("audit log consumers shutdown gracefully")
}
