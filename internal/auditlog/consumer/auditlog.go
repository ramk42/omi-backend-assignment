package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/nats-io/nats.go"
	"github.com/ramk42/omi-backend-assignment/internal/auditlog"
	"github.com/rs/zerolog/log"
	"time"
)

type AuditLog struct {
	auditLogUsecase auditlog.Usecase
	natsConn        *nats.Conn
}

func NewAuditLog(auditEventRepo auditlog.Usecase, natsConn *nats.Conn) *AuditLog {
	return &AuditLog{auditLogUsecase: auditEventRepo, natsConn: natsConn}
}

func (a *AuditLog) Start(ctx context.Context) error {
	sub, err := a.natsConn.QueueSubscribe("audit_logs", "audit_workers", func(msg *nats.Msg) {
		var event *auditlog.Model
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal audit event")
			return
		}
		err = a.auditLogUsecase.Push(ctx, event)
		if err != nil {
			log.Error().Err(err).Msg("failed to push audit event to usecase")
			if errors.Is(err, auditlog.ErrAuditLogBufferFull) {
				a.retryUntilBufferAvailable(ctx, event)
			}
			return
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to subscribe to NATS topic")
		return err
	}
	defer sub.Unsubscribe()

	log.Info().Msg("subscribed to NATS topic")
	select {
	case <-ctx.Done():
		log.Info().Msg("application shutting down, stopping audit log consumer")
		a.auditLogUsecase.Close()
		return nil
	}
}

func (a *AuditLog) retryUntilBufferAvailable(ctx context.Context, event *auditlog.Model) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := a.auditLogUsecase.Push(ctx, event)
			if err == nil {
				log.Info().Msg("audit log successfully pushed after buffer recovery")
				return
			}

			if errors.Is(err, auditlog.ErrAuditLogBufferFull) {
				log.Warn().Msg("audit log buffer still full, retrying...")
				continue
			}

			log.Error().Err(err).Msg("failed to push audit event, stopping retry")
			return

		case <-ctx.Done():
			log.Warn().Msg("application shutting down, aborting buffer retry")
			return
		}
	}
}
