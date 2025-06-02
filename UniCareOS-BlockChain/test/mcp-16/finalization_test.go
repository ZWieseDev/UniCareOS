package mcp16_test

import (
	"testing"
	"os"
	"time"
	"encoding/json"
	"encoding/base64"
	"fmt"

	"crypto/ed25519"
	"unicareos/core/block"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/google/uuid"
)

func init() {
	os.Setenv("MEDICAL_SCHEMA_PATH", "/mnt/e/AeonChain/UniCareOS/UniCareOS-BlockChain/core/validation/schemas/medical_record_schema_v1.json")
}

func loadFinalizerPrivateKey(path string) ed25519.PrivateKey {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		panic(err)
	}
	return ed25519.PrivateKey(decoded)
}

// MockAuditLogger is a mock implementation of the AuditLogger interface
type MockAuditLogger struct {
	mock.Mock
}

func (m *MockAuditLogger) LogFinalization(txID string, status block.FinalizationStatus, reason string) error {
	args := m.Called(txID, status, reason)
	return args.Error(0)
}

func TestFinalizer_FinalizeEvent(t *testing.T) {
	// Load a valid signed record from JSON
	data, err := os.ReadFile("../../signed_record.json")
	if err != nil {
		t.Fatalf("failed to read signed_record.json: %v", err)
	}
	type SignedRecord struct {
		Record        map[string]interface{} `json:"record"`
		Signature     string                 `json:"signature"`
		WalletAddress string                 `json:"walletAddress"`
	}
	var sr SignedRecord
	if err := json.Unmarshal(data, &sr); err != nil {
		t.Fatalf("failed to unmarshal signed_record.json: %v", err)
	}
	submission := block.MedicalRecordSubmission{
		Record:              sr.Record,
		Signature:           sr.Signature,
		WalletAddress:       sr.WalletAddress,
		SubmissionTimestamp: time.Now(),
	}
	submissionBytes, err := json.Marshal(submission)
	if err != nil {
		t.Fatalf("failed to marshal submission: %v", err)
	}
	// Load the finalizer private key from file
	privKey := loadFinalizerPrivateKey("../../finalizer_private.key")
	pubKey := privKey.Public().(ed25519.PublicKey)
	pubKeyStr := base64.StdEncoding.EncodeToString(pubKey)

	// Set up mock audit logger
	mockAudit := new(MockAuditLogger)
	mockAudit.On("LogFinalization", mock.Anything, block.FinalizationStatusFinalized, "finalization_successful").Return(nil)

	// Set up finalizer with authorized public key
	authorizedFinalizers := []string{pubKeyStr}
	finalizer := block.NewFinalizer(authorizedFinalizers, mockAudit, privKey)

	// Marshal the record in an envelope for schema validation
	envelope := map[string]interface{}{
		"record": sr.Record,
	}
	submissionBytes, err = json.Marshal(envelope)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}
	tx := &block.FinalizeEventTx{
		TxID:                  uuid.New().String(),
		SubmitMedicalRecordTx: submissionBytes,
		Status:                block.FinalizationStatusPending,
		Block:                 block.BlockReference{BlockHash: "testhash", Epoch: 1},
		Timestamp:             time.Now().UTC(),
	}
	// Set a valid signature for the top-level test
	signedData := append([]byte(tx.TxID), []byte(tx.Block.BlockHash)...)
	signature := ed25519.Sign(privKey, signedData)
	tx.FinalizerSignature = block.EncodeToBase64(signature)

	// Use tx in the test
	err = finalizer.FinalizeEvent(tx, pubKeyStr)
	assert.NoError(t, err)
	assert.Equal(t, block.FinalizationStatusFinalized, tx.Status)
	if tx.Status == block.FinalizationStatusFinalized {
		// Print in blue
		fmt.Printf("\033[1;34mRecord finalized! TxID: %s\033[0m\n", tx.TxID)
	}
	mockAudit.AssertExpectations(t)

	// Load the finalizer private key from file
	privKey = loadFinalizerPrivateKey("../../finalizer_private.key")
	pubKey = privKey.Public().(ed25519.PublicKey)
	pubKeyStr = base64.StdEncoding.EncodeToString(pubKey)
	// Use base64 strings for authorizedFinalizers
	authorizedFinalizers = []string{pubKeyStr}

	t.Run("Success", func(t *testing.T) {
		mockAudit := new(MockAuditLogger)
		mockAudit.On("LogFinalization", mock.Anything, block.FinalizationStatusFinalized, "finalization_successful").Return(nil)

		finalizer := block.NewFinalizer(authorizedFinalizers, mockAudit, privKey)

		// Create a test submission with valid UUID
		// Load the full, valid signed record from JSON
		data, err := os.ReadFile("../../signed_record.json")
		if err != nil {
			t.Fatalf("failed to read signed_record.json: %v", err)
		}
		type SignedRecord struct {
			Record        map[string]interface{} `json:"record"`
			Signature     string                 `json:"signature"`
			WalletAddress string                 `json:"walletAddress"`
		}
		var sr SignedRecord
		if err := json.Unmarshal(data, &sr); err != nil {
			t.Fatalf("failed to unmarshal signed_record.json: %v", err)
		}
		// Ensure recordId is a valid UUID v4
		recordId := uuid.New().String()
		sr.Record["recordId"] = recordId
		// Marshal the record in an envelope for schema validation
		envelope := map[string]interface{}{
			"record": sr.Record,
		}
		submissionBytes, err := json.Marshal(envelope)
		if err != nil {
			t.Fatalf("failed to marshal envelope: %v", err)
		}
		blockRef := block.BlockReference{
			BlockHash: "testhash",
			Epoch:     1,
		}
		// Create and sign the transaction with a valid UUID v4 txID
		txID := uuid.New().String()
		signedData := append([]byte(txID), []byte(blockRef.BlockHash)...)
		signature := ed25519.Sign(privKey, signedData)
		tx := &block.FinalizeEventTx{
			TxID:                 txID,
			SubmitMedicalRecordTx: submissionBytes,
			FinalizerSignature:   block.EncodeToBase64(signature),
			Block:                blockRef,
			Timestamp:            time.Now().UTC(),
			Status:               block.FinalizationStatusPending,
		}
		err = finalizer.FinalizeEvent(tx, pubKeyStr)
		assert.NoError(t, err)
		assert.Equal(t, block.FinalizationStatusFinalized, tx.Status)
		mockAudit.AssertExpectations(t)
	})

	t.Run("UnauthorizedFinalizer", func(t *testing.T) {
		finalizer := block.NewFinalizer(authorizedFinalizers, nil, privKey)
		tx := &block.FinalizeEventTx{}

		// Pass base64 string for unauthorized finalizer
		err := finalizer.FinalizeEvent(tx, "unauthorized-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized finalizer")
	})

	t.Run("InvalidTransaction", func(t *testing.T) {
		finalizer := block.NewFinalizer(authorizedFinalizers, nil, privKey)
		tx := &block.FinalizeEventTx{
			TxID: "", // Invalid empty TxID
		}

		err := finalizer.FinalizeEvent(tx, pubKeyStr)
		assert.Error(t, err)
	})

	t.Run("Atomicity_NoPartialMutationOnFailure", func(t *testing.T) {
		// Setup: valid tx but with invalid payload (simulate schema validation failure)
		mockAudit := new(MockAuditLogger)
		// Should NOT be called
		mockAudit.On("LogFinalization", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		finalizer := block.NewFinalizer(authorizedFinalizers, mockAudit, privKey)

		// Create a valid tx but with an invalid payload (e.g., missing required field)
		badRecord := map[string]interface{}{
			"someField": "not a valid schema field",
		}
		submissionBytes, err := json.Marshal(badRecord)
		assert.NoError(t, err)

		txID := uuid.New().String()
		blockRef := block.BlockReference{BlockHash: "testhash", Epoch: 1}
		signedData := append([]byte(txID), []byte(blockRef.BlockHash)...)
		signature := ed25519.Sign(privKey, signedData)
		originalSig := block.EncodeToBase64(signature)

		tx := &block.FinalizeEventTx{
			TxID:                 txID,
			SubmitMedicalRecordTx: submissionBytes,
			FinalizerSignature:   originalSig,
			Block:                blockRef,
			Timestamp:            time.Now().UTC(),
			Status:               block.FinalizationStatusPending,
		}

		// Save original state
		origStatus := tx.Status
		origSignature := tx.FinalizerSignature

		// Should fail validation (bad payload)
		err = finalizer.FinalizeEvent(tx, pubKeyStr)
		assert.Error(t, err, "Should error due to invalid payload")
		assert.Equal(t, origStatus, tx.Status, "Tx status should not change on failure")
		assert.Equal(t, origSignature, tx.FinalizerSignature, "Signature should not change on failure")

		// Ensure audit log was NOT called
		mockAudit.AssertNotCalled(t, "LogFinalization", mock.Anything, mock.Anything, mock.Anything)
	})
}
