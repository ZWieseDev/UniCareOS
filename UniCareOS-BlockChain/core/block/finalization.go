package block

import (
	"encoding/json"
	"errors"
	"time"
	"fmt"
    "encoding/base64"
    "crypto/ed25519"

	"unicareos/core/validation"
)

// Finalizer handles the finalization of medical record submissions
type Finalizer struct {
	authorizedFinalizers map[string]bool // Map of authorized finalizer public keys
	auditLog            AuditLogger
	privateKey          ed25519.PrivateKey // In-memory finalizer private key (never log)
}

// AuditLogger defines the interface for audit logging finalization events
type AuditLogger interface {
	LogFinalization(txID string, status FinalizationStatus, reason string) error
}

// NewFinalizer creates a new Finalizer with the given authorized finalizers
func NewFinalizer(authorizedFinalizers []string, auditLog AuditLogger, privateKey ed25519.PrivateKey) *Finalizer {
	finalizers := make(map[string]bool)
	for _, f := range authorizedFinalizers {
		finalizers[f] = true
	}
	return &Finalizer{
		authorizedFinalizers: finalizers,
		auditLog:            auditLog,
		privateKey:          privateKey,
	}
}

// FinalizeEvent handles the finalization of a medical record submission
// Implements atomic commit/rollback: all state changes are staged and only committed if all checks pass.
// If any error occurs, no shared state is mutated.
func (f *Finalizer) FinalizeEvent(
	tx *FinalizeEventTx,
	finalizerPubKey string,
) error {

	//fmt.Printf("\033[1;34m[FINALIZER] FinalizeEvent called for TxID: %s\033[0m\n", tx.TxID)

	// === ATOMIC FINALIZATION BEGIN ===
	// Stage all changes in local variables. Only commit if all checks pass.
	var (
		commitReady = false
		commitReason = ""
		commitStatus = FinalizationStatusFinalized
		commitAuditLog = f.auditLog
		commitSignature string
	)
	// 1. Validate the finalizer is authorized (NO STATE MUTATION)
	if !f.isAuthorized(finalizerPubKey) {
	
		return errors.New("unauthorized finalizer")
	} else {
	
	}
	// 2. Decode the base64 public key for cryptographic use (NO STATE MUTATION)
	pubKeyBytes, err := base64.StdEncoding.DecodeString(finalizerPubKey)
	if err != nil {
	
		return errors.New("invalid base64 public key")
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
	
		return errors.New("invalid public key length")
	}

	// 3. Stage signature if needed (NO STATE MUTATION)
	if f.privateKey == nil {
	
	}
	if f.privateKey != nil && tx.FinalizerSignature == "" && tx.TxID != "" && tx.Block.BlockHash != "" {
		msg := append([]byte(tx.TxID), []byte(tx.Block.BlockHash)...)
	
		sig := ed25519.Sign(f.privateKey, msg)
	
		commitSignature = base64.StdEncoding.EncodeToString(sig)
	}
	// 4. Validate the transaction using raw bytes (NO STATE MUTATION)
	validateSig := tx.FinalizerSignature
	if commitSignature != "" {
		validateSig = commitSignature // Use staged signature for validation
	}
	origSig := tx.FinalizerSignature
	tx.FinalizerSignature = validateSig
	if err := tx.Validate(pubKeyBytes); err != nil {
		tx.FinalizerSignature = origSig // rollback staged sig
	
		return err
	}
	tx.FinalizerSignature = origSig // restore

	// 5. Parse the submitted medical record envelope (NO STATE MUTATION)
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(tx.SubmitMedicalRecordTx, &envelope); err != nil {
	
		return err
	}
	recordPayload := envelope["record"]

	if err := validation.ValidateMedicalPayload(recordPayload); err != nil {
	
		return err
	}
	// === All checks passed: ready to commit ===
	commitReady = true
	commitReason = "finalization_successful"

	// --- COMMIT ATOMICALLY ---
	if commitReady {
		if commitSignature != "" {
			tx.FinalizerSignature = commitSignature
		}
		tx.Finalize()
		tx.Status = commitStatus
		// Print in blue when finalized
		fmt.Printf("\033[1;34m[FINALIZER] Record finalized! TxID: %s [atomic commit]\033[0m\n", tx.TxID)
		if commitAuditLog != nil {
			_ = commitAuditLog.LogFinalization(tx.TxID, tx.Status, commitReason)
		}
		return nil
	}
	// --- If we ever reach here, rollback ---

	return errors.New("atomic finalization failed")
}

// isAuthorized checks if the finalizer is in the authorized list
func (f *Finalizer) isAuthorized(finalizerPubKey string) bool {
	return f.authorizedFinalizers[finalizerPubKey]
}

// FinalizationResult represents the result of a finalization attempt
type FinalizationResult struct {
	TxID      string           `json:"txID"`
	Status    FinalizationStatus `json:"status"`
	BlockHash string           `json:"blockHash,omitempty"`
	Error     string           `json:"error,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
}
