package mcp20

import (
	"testing"
	"time"
	"github.com/stretchr/testify/require"
	"unicareos/core/block"
	"unicareos/types/ids"
)


import "os"
import "fmt"

func TestRevisionOfFinalizedEventAllowed(t *testing.T) {
	cwd, _ := os.Getwd()
	fmt.Println("DEBUG: Current working directory:", cwd)

	require.NoError(t, block.ReloadAuthorizedWallets())
	// Debug: authorized wallets will be printed from the block package ReloadAuthorizedWallets()

	// Create a new block
	b := &block.Block{
		Events: []block.ChainedEvent{},
	}

	// Use the same structure and values as signed_record.json for evtA
	evtA := block.ChainedEvent{
		RecordID:   "b0f3e7a4-1e8b-4c0e-bc8d-7c9b6a6c2e4f",
		EventID:    ids.NewID([]byte("evtA")),
		EventType:  "lab_result",
		Timestamp:  time.Now(),
		Finalized:  true, // Mark as finalized
		Description: "Original finalized record",
		PatientID:  "YWJjZGVmZ2hpamtsbW5vcA==",
		ProviderID: "encrypted-provider-id",
	}
	b.Events = append(b.Events, evtA)

	// Submit a revision (evtB) referencing evtA, using same structure as signed_record.json
	sub := block.MedicalRecordSubmission{
		Record: map[string]interface{}{
			"consentStatus": "granted",
			"dataProvenance": "hospital-system",
			"docHash": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			"encryptionContext": map[string]interface{}{
				"algorithm": "AES-GCM",
				"iv": "abcdefghijklmnop1234",
				"tag": "ZYXWVUTSRQPONMLK9876",
			},
			"issuedAt": "2025-05-25T15:00:00Z",
			"patientDID": "did:example:123456789abcdefghi",
			"patientId": "YWJjZGVmZ2hpamtsbW5vcA==",
			"payloadSignature": "QHOgzrGDgCLeepAAdwK5gPEP5E2yyPdqDGueTN4S8gOKIaa+dwJs+z11ghspAG3NsJIMOdRAX2+i1rN/WwB0Bg==",
			"providerId": "encrypted-provider-id",
			"recordId": "evtB-revision",
			"recordType": "lab_result",
			"retentionPolicy": "standard",
			"schemaVersion": "1.0",
			"signedBy": "test_wallet",
		},
		Signature:     "g78Li1R4Rlxm5rqW0ZyvA+sSbBFhUxOqicBeeuyJ7yxCyjTPTd3GqEcUv1B/ZzuiboMBqkXBkhtHHw1KuogYCA==",
		WalletAddress: "test_wallet",
		RevisionOf:    evtA.EventID.String(),
		SubmissionTimestamp: time.Now(),
	}

	receipt, err := block.SubmitRecordToBlock(sub, b)
	require.NoError(t, err, "should allow revision of finalized event")
	require.Equal(t, "pending", receipt.Status)

	// Check that evtB is appended and points to evtA
	found := false
	for _, evt := range b.Events {
		if evt.RecordID == "evtB-revision" && evt.RevisionOf == evtA.EventID.String() {
			found = true
		}
	}
	require.True(t, found, "evtB should be present and reference evtA")
}


func TestRevisionOfNonexistentEventBlocked(t *testing.T) {
	b := &block.Block{Events: []block.ChainedEvent{}}

	sub := block.MedicalRecordSubmission{
		Record: map[string]interface{}{
			"recordId": "evtX",
		},
		Signature:     "dummy-signature",
		WalletAddress: "providerID_wallet",
		RevisionOf:    "nonexistent-event-id",
		SubmissionTimestamp: time.Now(),
	}

	receipt, err := block.SubmitRecordToBlock(sub, b)
	require.Error(t, err, "should block revision of nonexistent event")
	require.Equal(t, "failed", receipt.Status)
}
