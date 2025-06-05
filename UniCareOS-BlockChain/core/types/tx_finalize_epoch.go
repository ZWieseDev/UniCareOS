package types

import (
	"time"
	"fmt"
)

// FinalizeEpochTx represents a transaction to finalize an epoch
// and make all blocks/events within it immutable.
type FinalizeEpochTx struct {
	TxID               string    `json:"txID"`
	EpochNumber        uint64    `json:"epochNumber"`
	FinalizerSignature string    `json:"finalizerSignature"`
	EpochSummaryHash   string    `json:"epochSummaryHash,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
	Status             string    `json:"status"` // finalized|failed|duplicate|unauthorized
	AuditLogID         string    `json:"auditLogID,omitempty"`
}

// NewFinalizeEpochTx creates a new FinalizeEpochTx instance
func NewFinalizeEpochTx(epochNumber uint64, finalizerSignature, epochSummaryHash string) *FinalizeEpochTx {
	return &FinalizeEpochTx{
		TxID:               "", // Should be set by the caller or assigned after hashing
		EpochNumber:        epochNumber,
		FinalizerSignature: finalizerSignature,
		EpochSummaryHash:   epochSummaryHash,
		Timestamp:          time.Now().UTC(),
		Status:             "pending",
		AuditLogID:         "",
	}
}

// Validate checks the validity of the FinalizeEpochTx fields
func (tx *FinalizeEpochTx) Validate() error {
	if tx.EpochNumber == 0 {
		return fmt.Errorf("epochNumber must be non-zero")
	}
	if tx.FinalizerSignature == "" {
		return fmt.Errorf("finalizerSignature is required")
	}
	// TODO: Add cryptographic signature validation here
	// Example: verifySignature(tx.FinalizerSignature, ...)
	return nil
}
// TransactionReceipt for epoch finalization
// (could be reused from your existing receipt type, but included here for clarity)
type EpochFinalizationReceipt struct {
	TxID        string    `json:"txID"`
	Status      string    `json:"status"`
	EpochNumber uint64    `json:"epochNumber"`
	Timestamp   time.Time `json:"timestamp"`
	Errors      []string  `json:"errors,omitempty"`
}
