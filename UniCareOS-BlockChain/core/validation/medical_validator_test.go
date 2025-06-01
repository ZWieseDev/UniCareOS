package validation

import (
	"testing"
)

func validPayload() []byte {
	return []byte(`{
  "recordId": "123e4567-e89b-12d3-a456-426614174000",
  "patientId": "HOSP12345",
  "patientDID": "did:example:123456abcdef",
  "providerID": "PROV123",
  "schemaVersion": "1.0",
  "recordType": "lab_result",
  "docHash": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "issuedAt": "2025-05-22T18:00:00Z",
  "signedBy": "PROV123",
  "consentStatus": "granted",
  "dataProvenance": "hospital-system",
  "retentionPolicy": "7 years",
  "encryptionContext": {
    "algorithm": "AES-GCM",
    "iv": "YWJjZGVmZ2hpamtsbW5vcA==",
    "tag": "YWJjZGVmZ2hpamtsbW5vcA=="
  },
  "payloadSignature": "YWJjZGVmZ2hpamtsbW5vcA=="
}`)
}

func TestValidateMedicalPayload_Valid(t *testing.T) {
	err := ValidateMedicalPayload(validPayload())
	if err != nil {
		t.Errorf("Expected valid payload, got error: %v", err)
	}
}

func TestValidateMedicalPayload_MissingField(t *testing.T) {
	payload := []byte(`{
  "recordId": "123e4567-e89b-12d3-a456-426614174000"
}`)
	err := ValidateMedicalPayload(payload)
	if err == nil {
		t.Errorf("Expected error for missing fields, got nil")
	}
}

func TestValidateMedicalPayload_InvalidDocHash(t *testing.T) {
	payload := []byte(`{
  "recordId": "123e4567-e89b-12d3-a456-426614174000",
  "patientId": "HOSP12345",
  "patientDID": "did:example:123456abcdef",
  "providerID": "PROV123",
  "schemaVersion": "1.0",
  "recordType": "lab_result",
  "docHash": "notavalidhexhash",
  "issuedAt": "2025-05-22T18:00:00Z",
  "signedBy": "PROV123",
  "consentStatus": "granted",
  "dataProvenance": "hospital-system",
  "retentionPolicy": "7 years",
  "encryptionContext": {
    "algorithm": "AES-GCM",
    "iv": "YWJjZGVmZ2hpamtsbW5vcA==",
    "tag": "YWJjZGVmZ2hpamtsbW5vcA=="
  },
  "payloadSignature": "YWJjZGVmZ2hpamtsbW5vcA=="
}`)
	err := ValidateMedicalPayload(payload)
	if err == nil {
		t.Errorf("Expected error for invalid docHash, got nil")
	}
}

func TestValidateMedicalPayload_InvalidTimestamp(t *testing.T) {
	payload := []byte(`{
  "recordId": "123e4567-e89b-12d3-a456-426614174000",
  "patientId": "HOSP12345",
  "patientDID": "did:example:123456abcdef",
  "providerID": "PROV123",
  "schemaVersion": "1.0",
  "recordType": "lab_result",
  "docHash": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "issuedAt": "not-a-date",
  "signedBy": "PROV123",
  "consentStatus": "granted",
  "dataProvenance": "hospital-system",
  "retentionPolicy": "7 years",
  "encryptionContext": {
    "algorithm": "AES-GCM",
    "iv": "YWJjZGVmZ2hpamtsbW5vcA==",
    "tag": "YWJjZGVmZ2hpamtsbW5vcA=="
  },
  "payloadSignature": "YWJjZGVmZ2hpamtsbW5vcA=="
}`)
	err := ValidateMedicalPayload(payload)
	if err == nil {
		t.Errorf("Expected error for invalid issuedAt, got nil")
	}
}
