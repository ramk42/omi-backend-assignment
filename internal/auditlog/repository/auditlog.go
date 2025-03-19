package repository

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ramk42/omi-backend-assignment/internal/auditlog"
	"github.com/rs/zerolog/log"
	"time"
)

type AuditLog struct {
	db *pgxpool.Pool
}

func NewAuditEventRepository(db *pgxpool.Pool) *AuditLog {
	return &AuditLog{db: db}
}

func (r *AuditLog) Insert(ctx context.Context, events []*auditlog.Model) error {
	if len(events) == 0 {
		log.Warn().Msg("no audit events to insert")
		return nil
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := `
		INSERT INTO audit_logs (id, spec_version, source, type, subject, timestamp, actor, action, resource, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	batch := &pgx.Batch{}

	for _, event := range events {
		// Ensure event has a valid ID
		if event.ID == "" {
			event.ID = uuid.NewString()
		}

		batch.Queue(query,
			event.ID,
			event.SpecVersion,
			event.Source,
			event.Type,
			event.Subject,
			event.Timestamp,
			event.Actor,
			event.Action,
			event.Resource,
			event.Metadata,
		)
	}

	br := r.db.SendBatch(timeoutCtx, batch)
	defer br.Close()

	_, err := br.Exec()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Error().Err(err).Msg("database timeout exceeded")
			return auditlog.ErrDatabaseUnavailable
		}
		var pgErr *pgconn.ConnectError
		if errors.As(err, &pgErr) {
			log.Error().Err(err).Msg("connection lost to the database")
			return auditlog.ErrDatabaseUnavailable
		}
		return err
	}

	log.Info().Msgf("Successfully inserted %d audit events", len(events))
	return nil
}
