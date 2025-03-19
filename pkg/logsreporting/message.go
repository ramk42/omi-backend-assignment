package logsreporting

import (
	"time"
)

type AuditLog struct {
	SpecVersion string            `json:"spec_version"`
	ID          string            `json:"id"`
	Source      string            `json:"source"`
	Type        string            `json:"type"`
	Subject     string            `json:"subject"`
	Timestamp   time.Time         `json:"timestamp"`
	Actor       map[string]string `json:"actor"`
	Action      string            `json:"action"`
	Resource    map[string]any    `json:"resource"`
	Metadata    map[string]string `json:"metadata"`
}
