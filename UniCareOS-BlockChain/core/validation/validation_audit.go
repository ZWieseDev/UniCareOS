package validation

import (
	"log"
	"os"
	"sync"
)

var auditOnce sync.Once
var auditLogger *log.Logger

func getAuditLogger() *log.Logger {
	auditOnce.Do(func() {
		f, err := os.OpenFile("validation_audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open audit log: %v", err)
		}
		auditLogger = log.New(f, "[AUDIT] ", log.LstdFlags|log.LUTC)
	})
	return auditLogger
}

// AuditValidationError logs validation errors (without PHI) to a file
func AuditValidationError(context, errMsg string) {
	logger := getAuditLogger()
	logger.Printf("%s | %s", context, errMsg)
}
