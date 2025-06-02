package mcp12_test

import (
	"testing"
	"time"
	"unicareos/core/block"
)

func TestSignatureVerificationFailure(t *testing.T) {
	t.Setenv("MEDICAL_SCHEMA_PATH", "../../core/validation/schemas/medical_record_schema_v1.json")
	b := &block.Block{}
	submission := block.MedicalRecordSubmission{
		Record: map[string]interface{}{
			"recordId": "3fa85f64-5717-4562-b3fc-2c963f66afa8",
			"patientId": "YWJj",
			"patientDID": "did:test:abc",
			"providerId": "cHJvdjEyMw==",
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
		Signature:           "bad_signature",
		WalletAddress:       "providerID_wallet",
		SubmissionTimestamp: time.Now(),
	}
	receipt, err := block.SubmitRecordToBlock(submission, b)
	if err == nil || receipt.Status != "failed" {
		t.Errorf("Expected signature verification to fail, got status: %v, error: %v", receipt.Status, err)
	}
	t.Logf("Audit log: %+v", b.AuditLog)
	if len(b.AuditLog) == 0 || b.AuditLog[len(b.AuditLog)-1].Status != "invalid_signature" {
		t.Errorf("Expected audit log entry with status 'invalid_signature', got %+v", b.AuditLog)
	}
}

func TestSignatureVerificationSuccess(t *testing.T) {
	t.Setenv("MEDICAL_SCHEMA_PATH", "../../core/validation/schemas/medical_record_schema_v1.json")
	b := &block.Block{}
	submission := block.MedicalRecordSubmission{
		Record: map[string]interface{}{
			"recordId": "3fa85f64-5717-4562-b3fc-2c963f66afa7", // valid UUID v4
			"patientId": "YWJj",
			"patientDID": "did:test:abc",
			"providerId": "cHJvdjEyMw==",
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
		WalletAddress:       "providerID_wallet",
		SubmissionTimestamp: time.Now(),
	}
	receipt, err := block.SubmitRecordToBlock(submission, b)
	if err != nil || receipt.Status != "pending" {
		t.Errorf("Expected signature verification to succeed, got status: %v, error: %v", receipt.Status, err)
	}
	if len(b.AuditLog) == 0 || b.AuditLog[len(b.AuditLog)-1].Status != "accepted" {
		t.Errorf("Expected audit log entry with status 'accepted', got %+v", b.AuditLog)
	}
}
