package logsreporting

import (
	"context"
)

type Producer interface {
	Publish(ctx context.Context, audiLogMsg *AuditLog) error
}
