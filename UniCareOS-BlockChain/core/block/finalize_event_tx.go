package block

import (
	"time"

	"errors"
	"encoding/json"
	"crypto/ed25519"
	"encoding/base64"
	"github.com/google/uuid"
)


// FinalizationStatus represents the status of a finalization attempt
type FinalizationStatus string

// Finalization status constants
const (
	// FinalizationStatusPending indicates the finalization is pending
	FinalizationStatusPending FinalizationStatus = "pending"
	// FinalizationStatusFinalized indicates successful finalization
	FinalizationStatusFinalized FinalizationStatus = "finalized"
	// FinalizationStatusFailed indicates finalization failed
	FinalizationStatusFailed FinalizationStatus = "failed"
	// FinalizationStatusDuplicate indicates a duplicate finalization attempt
	FinalizationStatusDuplicate FinalizationStatus = "duplicate"
)

// FinalizeEventTx represents a transaction to finalize a previously submitted medical record
type FinalizeEventTx struct {
	TxID                string                 `json:"txID"`
	SubmitMedicalRecordTx json.RawMessage       `json:"submitMedicalRecordTx"`
	FinalizerSignature  string                `json:"finalizerSignature"`
	EthosToken         string                `json:"ethosToken"`
	Block              BlockReference        `json:"block"`
	Timestamp          time.Time             `json:"timestamp"`
	Status             FinalizationStatus    `json:"status"`
	AuditLogID         string                `json:"auditLogId,omitempty"`
}

// MarshalCanonical serializes FinalizeEventTx to canonical JSON (deterministic field order, no extra whitespace)
func (tx *FinalizeEventTx) MarshalCanonical() ([]byte, error) {
	type canonicalFinalizeEventTx struct {
		TxID                string          `json:"txID"`
		SubmitMedicalRecordTx json.RawMessage `json:"submitMedicalRecordTx"`
		FinalizerSignature  string          `json:"finalizerSignature"`
		EthosToken          string          `json:"ethosToken"`
		Block               BlockReference `json:"block"`
		Timestamp           string         `json:"timestamp"`
		Status              FinalizationStatus `json:"status"`
		AuditLogID          string         `json:"auditLogId,omitempty"`
	}
	c := canonicalFinalizeEventTx{
		TxID: tx.TxID,
		SubmitMedicalRecordTx: tx.SubmitMedicalRecordTx,
		FinalizerSignature: tx.FinalizerSignature,
		EthosToken: tx.EthosToken,
		Block: tx.Block,
		Timestamp: tx.Timestamp.UTC().Format(time.RFC3339Nano),
		Status: tx.Status,
		AuditLogID: tx.AuditLogID,
	}
	return json.Marshal(c)
}


// BlockReference contains block metadata for finalization
type BlockReference struct {
	BlockHash string `json:"blockHash"`
	Epoch     uint64 `json:"epoch"`
}



// NewFinalizeEventTx creates a new FinalizeEventTx with the given parameters
func NewFinalizeEventTx(
	submitTx json.RawMessage,
	finalizerPubKey ed25519.PublicKey,
	signature []byte,
	ethosToken string,
	blockRef BlockReference,
) (*FinalizeEventTx, error) {
	if len(submitTx) == 0 {
		return nil, errors.New("submitMedicalRecordTx cannot be empty")
	}
	if len(signature) == 0 {
		return nil, errors.New("signature cannot be empty")
	}

	tx := &FinalizeEventTx{
		TxID:                 uuid.New().String(),
		SubmitMedicalRecordTx: submitTx,
		FinalizerSignature:   base64.StdEncoding.EncodeToString(signature),
		EthosToken:          ethosToken,
		Block:               blockRef,
		Timestamp:           time.Now().UTC(),
		Status:              FinalizationStatusPending,
	}

	return tx, nil
}

// Validate checks if the FinalizeEventTx is valid
func (tx *FinalizeEventTx) Validate(finalizerPubKey ed25519.PublicKey) error {
	if tx.TxID == "" {
		return errors.New("txID cannot be empty")
	}
	if len(tx.SubmitMedicalRecordTx) == 0 {
		return errors.New("submitMedicalRecordTx cannot be empty")
	}
	if tx.Block.BlockHash == "" {
		return errors.New("block hash cannot be empty")
	}
	if tx.Timestamp.IsZero() {
		return errors.New("timestamp cannot be zero")
	}

	// Verify the finalizer's signature
	sig, err := base64.StdEncoding.DecodeString(tx.FinalizerSignature)
	if err != nil {
		return errors.New("invalid signature encoding")
	}

	// The signed data should be the concatenation of txID and the block hash



	signedData := append([]byte(tx.TxID), []byte(tx.Block.BlockHash)...)
	if !ed25519.Verify(finalizerPubKey, signedData, sig) {
		return errors.New("invalid finalizer signature")
	}

	return nil
}

// Finalize marks the transaction as finalized
func (tx *FinalizeEventTx) Finalize() {
	tx.Status = FinalizationStatusFinalized
	tx.Timestamp = time.Now().UTC()
	tx.AuditLogID = "finalized"
}

// MarkFailed marks the transaction as failed with the given reason
func (tx *FinalizeEventTx) MarkFailed(reason string) {
	tx.Status = FinalizationStatusFailed
	// Store reason in audit log ID if not already set
	if tx.AuditLogID == "" {
		tx.AuditLogID = "failed:" + reason
	}
}
