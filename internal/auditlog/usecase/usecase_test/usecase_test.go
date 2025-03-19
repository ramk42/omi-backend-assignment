package usecase_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ramk42/omi-backend-assignment/internal/auditlog"
	"github.com/ramk42/omi-backend-assignment/internal/auditlog/repository"
	"github.com/ramk42/omi-backend-assignment/internal/auditlog/usecase"
	"github.com/ramk42/omi-backend-assignment/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func SetupTestDB(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()
	dbpool, err := pgxpool.New(ctx, env.MustGetEnv[string]("DATABASE_URL"))
	require.NoError(t, err, "Failed to connect to test database")

	// Clean up the table before each test
	_, err = dbpool.Exec(ctx, "TRUNCATE audit_logs RESTART IDENTITY;")
	require.NoError(t, err, "Failed to clean audit_logs table")

	return dbpool
}

func generateTestEvent() *auditlog.Model {
	return &auditlog.Model{
		ID:          uuid.NewString(),
		SpecVersion: "1.0",
		Source:      "backend.api",
		Type:        "audit.event",
		Subject:     "event:account:4eaa2b93-c0e2-4556-83a3-ecfbc7d60fa3",
		Timestamp:   time.Now().UTC(),
		Actor: map[string]string{
			"id": uuid.NewString(),
		},
		Action: "PATCH",
		Resource: map[string]any{
			"id":   "4eaa2b93-c0e2-4556-83a3-ecfbc7d60fa3",
			"type": "account",
			"attributes": map[string]string{
				"id":         "4eaa2b93-c0e2-4556-83a3-ecfbc7d60fa3",
				"type":       "account",
				"attributes": `{"name": "John Doe", "email": "john.doe@example.com"}`,
			},
		},
		Metadata: map[string]string{
			"request_id":      uuid.NewString(),
			"response_status": "204",
			"protocol":        "HTTP/1.1",
		},
	}
}

func TestAuditLog_ShouldInsertLogs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := SetupTestDB(t)
	defer db.Close()

	auditLogRepo := repository.NewAuditEventRepository(db)

	auditLogUsecase := usecase.NewAuditLog(ctx, auditLogRepo, 3, 1*time.Second)

	events := []*auditlog.Model{
		generateTestEvent(),
		generateTestEvent(),
		generateTestEvent(),
	}

	for _, e := range events {
		err := auditLogUsecase.Push(ctx, e)
		require.NoError(t, err, "Push should not return an error")
	}

	time.Sleep(1 * time.Second) // wait a little bit before querying the database
	var count int
	err := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM audit_logs").Scan(&count)
	require.NoError(t, err, "Failed to count rows in audit_logs")
	assert.Equal(t, 3, count, "Audit logs should have been inserted")

	cancel()
	auditLogUsecase.Close()
}

func TestAuditLog_Buffer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := SetupTestDB(t)
	defer db.Close()

	auditLogRepo := repository.NewAuditEventRepository(db)

	auditLogUsecase := usecase.NewAuditLog(ctx, auditLogRepo, 1, 5*time.Second)

	events := []*auditlog.Model{
		generateTestEvent(),
		generateTestEvent(),
		generateTestEvent(),
	}

	err := auditLogUsecase.Push(ctx, events[0])
	require.NoError(t, err, "first Push should not return an error")

	err = auditLogUsecase.Push(ctx, events[1])
	require.Error(t, err, "second Push should return an error due to full buffer")
	assert.Equal(t, auditlog.ErrAuditLogBufferFull, err, "Expected ErrAuditLogBufferFull")

	err = auditLogUsecase.Push(ctx, generateTestEvent())
	require.NoError(t, err, "after flush, Push should not return an error")

	cancel()
	auditLogUsecase.Close()
}
