package logsreporting

import (
	"context"
	"encoding/json"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"time"
)

type LogsReporting struct {
	natsConn *nats.Conn
}

func NewProducer(natsConn *nats.Conn) *LogsReporting {
	return &LogsReporting{
		natsConn: natsConn,
	}
}

func (l *LogsReporting) Publish(mainCtx context.Context, auditLogMsg *AuditLog) error {
	log.Debug().Msg("publishing audit logs...")

	data, err := json.Marshal(auditLogMsg)
	if err != nil {
		log.Err(err).Msg("error marshalling audit log")
		return err
	}

	const maxRetries = 10
	baseDelay := time.Second * 2

	for i := 0; i < maxRetries; i++ {
		// Wait for NATS to be connected before publishing
		if !l.natsConn.IsConnected() {
			log.Warn().Msg("NATS is disconnected, waiting for reconnection...")
			for !l.natsConn.IsConnected() {
				select {
				case <-time.After(time.Second): // Check connection every second
				case <-mainCtx.Done(): // Application shutdown detected
					log.Warn().Msg("publishing aborted due to application shutdown (mainCtx canceled)")
					return mainCtx.Err()
				}
			}
			log.Info().Msg("NATS reconnected, resuming publish")
		}

		err = l.natsConn.Publish("audit_logs", data)
		if err == nil {
			log.Info().Msg("audit log successfully published")
			return nil
		}

		log.Err(err).Msgf("⚠error publishing audit log, attempt %d/%d", i+1, maxRetries)

		waitTime := baseDelay * time.Duration(1<<i)
		select {
		case <-time.After(waitTime):
		case <-mainCtx.Done(): // Application shutdown detected
			log.Warn().Msg("publishing stopped due to application shutdown")
			return mainCtx.Err()
		}
	}

	log.Error().Msg("failed to publish audit log after multiple attempts")
	return err
}
