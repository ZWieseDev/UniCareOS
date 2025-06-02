package mcp12_test

import (
    "testing"
    "time"
    "unicareos/core/block"
)

func TestReplayProtection(t *testing.T) {
	b := &block.Block{}
	submission := block.MedicalRecordSubmission{
		Record: map[string]interface{}{
			"recordId": "123e4567-e89b-12d3-a456-426614174000",
			"patientId": "YWJj", // "abc" in base64
			"patientDID": "did:test:abc",
			"providerId": "cHJvdjEyMw==", // "prov123" in base64
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

	t.Setenv("MEDICAL_SCHEMA_PATH", "../../core/validation/medical_record_schema.json")

	// First submission should succeed
	receipt1, err1 := block.SubmitRecordToBlock(submission, b)
	if err1 != nil || receipt1.Status != "pending" {
		t.Fatalf("First submission failed unexpectedly: %v, receipt: %+v", err1, receipt1)
	}

	// Second submission with same recordId should fail (replay protection)
	submission2 := submission
	submission2.Signature = "YWJjZGVmZ2hpamtsbW5vcA==" // could be different, doesn't matter for replay
	submission2.SubmissionTimestamp = time.Now().Add(1 * time.Minute) // simulate later submission

	receipt2, err2 := block.SubmitRecordToBlock(submission2, b)
	if err2 == nil || receipt2.Status != "failed" {
		t.Fatalf("Expected replay protection to fail second submission, got: %v, receipt: %+v", err2, receipt2)
	}
	if len(b.AuditLog) < 2 || b.AuditLog[len(b.AuditLog)-1].Status != "duplicate" {
		t.Errorf("Expected last audit log entry to be 'duplicate', got %+v", b.AuditLog[len(b.AuditLog)-1])
	}
}
