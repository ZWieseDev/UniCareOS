package audit

import (
	"fmt"
	"time"
)

// AuditEvent represents a verification or authorization event.
type AuditEvent struct {
	Timestamp   time.Time
	EventType   string // e.g., "SignatureVerification", "EthosTokenVerification"
	EntityID    string // e.g., wallet address or token subject
	Result      string // e.g., "success", "failure"
	Reason      string // error message or reason code
	Metadata    map[string]string // any extra details
}

// AuditLogger is the interface for logging audit events.
type AuditLogger interface {
	LogEvent(event AuditEvent)
}

// StdoutAuditLogger is a simple implementation that logs to stdout.
type StdoutAuditLogger struct{}

func (l *StdoutAuditLogger) LogEvent(event AuditEvent) {
	fmt.Printf("[%s] [%s] Entity: %s, Result: %s, Reason: %s, Metadata: %+v\n",
		event.Timestamp.Format(time.RFC3339), event.EventType, event.EntityID, event.Result, event.Reason, event.Metadata)
}

// NewStdoutAuditLogger returns a new StdoutAuditLogger.
func NewStdoutAuditLogger() AuditLogger {
	return &StdoutAuditLogger{}
}
