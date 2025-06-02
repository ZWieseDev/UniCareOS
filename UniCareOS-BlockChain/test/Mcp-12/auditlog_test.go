package mcp12_test

import (
	"testing"
	"unicareos/core/block"
	"time"
)

func TestAuditLogEntryCreated(t *testing.T) {
	// TODO: Implement after audit log logic is ready
}

func TestAuditLogEntries(t *testing.T) {
	var b *block.Block
	b = &block.Block{}
	// Unauthorized wallet test
	unauthSubmission := block.MedicalRecordSubmission{
		Record: map[string]interface{}{
			"recordId": "unauth-123",
			"patientId": "YWJj",
			"patientDID": "did:test:abc",
			"providerID": "cHJvdjEyMw==",
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
		WalletAddress:       "unauthorized_wallet",
		SubmissionTimestamp: time.Now(),
	}
	_, _ = block.SubmitRecordToBlock(unauthSubmission, b)
	if len(b.AuditLog) == 0 || b.AuditLog[len(b.AuditLog)-1].Status != "unauthorized" {
		t.Errorf("Expected audit log entry with status 'unauthorized' for unauthorized wallet, got %+v", b.AuditLog)
	}

	b = &block.Block{}
	t.Setenv("MEDICAL_SCHEMA_PATH", "../../core/validation/schemas/medical_record_schema_v1.json")

	// 1. Validation failure
	invalidSubmission := block.MedicalRecordSubmission{
		Record: map[string]interface{}{
			// Missing required fields, e.g., no recordId
			"recordId": "valfail-4567-e89b-12d3-a456-426614174000",
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
	_, _ = block.SubmitRecordToBlock(invalidSubmission, b)

	if len(b.AuditLog) == 0 || b.AuditLog[len(b.AuditLog)-1].Status != "failed" {
		t.Errorf("Expected audit log entry with status 'failed' for validation error, got %+v", b.AuditLog)
	}

	// 2. Signature failure (use fully valid record, but invalid signature)
	validSigSubmission := block.MedicalRecordSubmission{
		Record: map[string]interface{}{
			"recordId": "3fa85f64-5717-4562-b3fc-2c963f66afa6", 
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

	_, _ = block.SubmitRecordToBlock(validSigSubmission, b)

	if len(b.AuditLog) < 2 || b.AuditLog[len(b.AuditLog)-1].Status != "invalid_signature" {
		t.Errorf("Expected audit log entry with status 'invalid_signature' for signature error, got %+v", b.AuditLog)
	}
}
