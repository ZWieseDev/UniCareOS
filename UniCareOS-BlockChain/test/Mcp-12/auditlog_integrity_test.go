package mcp12

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
	"unicareos/core/block"
)

// Helper to recompute the hash for an AuditLogEntry (excluding EntryHash)
func computeEntryHash(entry block.AuditLogEntry) string {
	h := sha256.New()
	h.Write([]byte(entry.EventID))
	h.Write([]byte(entry.SubmittedBy))
	h.Write([]byte(entry.Timestamp.Format(time.RFC3339Nano)))
	h.Write([]byte(entry.Status))
	h.Write([]byte(entry.Reason))
	h.Write([]byte(entry.PrevHash))
	return hex.EncodeToString(h.Sum(nil))
}

// Verifies the audit log hash chain
func verifyAuditLogChain(auditLog []block.AuditLogEntry) bool {
	for i, entry := range auditLog {
		if computeEntryHash(entry) != entry.EntryHash {
			return false // Entry hash mismatch (tampered)
		}
		if i > 0 && entry.PrevHash != auditLog[i-1].EntryHash {
			return false // Chain broken
		}
	}
	return true
}

func TestAuditLogIntegrity(t *testing.T) {
	b := &block.Block{}
	timestamp := time.Now()

	// Log 3 events
	block.LogSubmissionTrace(b, "evt1", "wallet1", "accepted", "ok", timestamp)
	block.LogSubmissionTrace(b, "evt2", "wallet2", "failed", "bad sig", timestamp.Add(time.Second))
	block.LogSubmissionTrace(b, "evt3", "wallet3", "duplicate", "already exists", timestamp.Add(2*time.Second))

	if !verifyAuditLogChain(b.AuditLog) {
		t.Fatal("Audit log chain should be valid for untampered log")
	}

	// Tamper with an entry
	b.AuditLog[1].Reason = "tampered reason"
	if verifyAuditLogChain(b.AuditLog) {
		t.Fatal("Audit log chain verification should fail after tampering")
	}
}
