package mcp16_test

import (
	"testing"
	"time"
	"encoding/json"
	"crypto/ed25519"
	"crypto/rand"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"unicareos/core/block"
)

func TestFinalizeEventTx(t *testing.T) {
	// Generate test keys
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	t.Run("NewFinalizeEventTx", func(t *testing.T) {
		submitTx := json.RawMessage(`{"recordId":"` + uuid.New().String() + `"}`)
		blockRef := block.BlockReference{
			BlockHash: "testhash",
			Epoch:     1,
		}
		txID := uuid.New().String()
		signedData := append([]byte(txID), []byte(blockRef.BlockHash)...)
		signature := ed25519.Sign(privKey, signedData)
		tx, err := block.NewFinalizeEventTx(
			submitTx,
			pubKey,
			signature,
			"",
			blockRef,
		)
		require.NoError(t, err)
		require.NotEmpty(t, tx.TxID)
		assert.Equal(t, block.FinalizationStatusPending, tx.Status)
	})

	t.Run("Validate_Success", func(t *testing.T) {
		submitTx := json.RawMessage(`{"recordId":"` + uuid.New().String() + `"}`)
		blockRef := block.BlockReference{
			BlockHash: "testhash",
			Epoch:     1,
		}
		tx, err := block.NewFinalizeEventTx(
			submitTx,
			pubKey,
			[]byte("dummy"), // use dummy non-nil signature at construction
			"",
			blockRef,
		)
		require.NoError(t, err)
		tx.Timestamp = time.Now().UTC()
		// Now sign using the actual tx.TxID
		signedData := append([]byte(tx.TxID), []byte(blockRef.BlockHash)...)
		signature := ed25519.Sign(privKey, signedData)
		tx.FinalizerSignature = block.EncodeToBase64(signature)
		// Validate after setting the signature
		err = tx.Validate(pubKey)
		assert.NoError(t, err)
	})

	t.Run("Validate_InvalidSignature", func(t *testing.T) {
		tx := &block.FinalizeEventTx{
			TxID:                "test-tx",
			SubmitMedicalRecordTx: json.RawMessage(`{"recordId":"123e4567-e89b-42d3-a4aa-426614174111"}`),
			FinalizerSignature:  "invalid-signature",
			Block: block.BlockReference{
				BlockHash: "testhash",
				Epoch:     1,
			},
			Timestamp: time.Now().UTC(),
			Status:    block.FinalizationStatusPending,
		}

		err := tx.Validate(pubKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature encoding")
	})

	t.Run("Finalize", func(t *testing.T) {
		tx := &block.FinalizeEventTx{
			Status: block.FinalizationStatusPending,
		}

		tx.Finalize()
		assert.Equal(t, block.FinalizationStatusFinalized, tx.Status)
	})

	t.Run("MarkFailed", func(t *testing.T) {
		tx := &block.FinalizeEventTx{
			Status: block.FinalizationStatusPending,
		}

		tx.MarkFailed("test_reason")
		assert.Equal(t, block.FinalizationStatusFailed, tx.Status)
		assert.Equal(t, "failed:test_reason", tx.AuditLogID)
	})
}
