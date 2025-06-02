package mcp12_test

import (
	"testing"
	"time"
	"unicareos/core/block"
)

func TestBlockUpdateOnSubmission(t *testing.T) {
	t.Setenv("MEDICAL_SCHEMA_PATH", "../../core/validation/medical_record_schema.json")
	b := &block.Block{}
	submission := block.MedicalRecordSubmission{
		Record: map[string]interface{}{
			"recordId": "123e4567-e89b-12d3-a456-426614174000",
			"patientId": "YWJj", // "abc" in base64
			"patientDID": "did:test:abc",
			"providerID": "cHJvdjEyMw==", // "prov123" in base64
			"schemaVersion": "1.0",
			"recordType": "lab_result",
			"docHash": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			"issuedAt": "2025-05-22T18:00:00Z",
			"signedBy": "prov123",
			"consentStatus": "granted",
			"dataProvenance": "hospital-system",
			"retentionPolicy": "7 years",
			"encryptionContext": map[string]interface{}{
				"algorithm": "AES-GCM",
				"iv": "YWJjZGVmZ2hpamtsbW5vcA==",
				"tag": "YWJjZGVmZ2hpamtsbW5vcA==",
			},
			"payloadSignature": "YWJjZGVmZ2hpamtsbW5vcA==",
		},
		Signature:           "YWJjZGVmZ2hpamtsbW5vcA==",
		WalletAddress:       "provider_wallet",
		SubmissionTimestamp: time.Now(),
	}

	receipt, err := block.SubmitRecordToBlock(submission, b)
	if err != nil || receipt.Status != "pending" {
		t.Fatalf("Expected success, got error: %v, receipt: %+v", err, receipt)
	}
	if len(b.Events) != 1 {
		t.Errorf("Expected 1 event in block, got %d", len(b.Events))
	}
	if len(b.AuditLog) != 1 || b.AuditLog[0].Status != "accepted" {
		t.Errorf("Expected 1 accepted audit log entry, got %+v", b.AuditLog)
	}
}
