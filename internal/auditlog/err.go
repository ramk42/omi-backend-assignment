package auditlog

import (
	"errors"
)

// Usecase errors
var (
	ErrAuditLogBufferFull = errors.New("audit log buffer is full")
	ErrAuditLogClosed     = errors.New("audit log is closed")
)

// ErrDatabaseUnavailable Repository errors
var (
	ErrDatabaseUnavailable = errors.New("database is not accessible")
)
