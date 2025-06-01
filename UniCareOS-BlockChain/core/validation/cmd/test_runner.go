package main

import (
	"fmt"
	"os"
	"unicareos/core/validation"
)

const yellow = "\033[33m"
const reset = "\033[0m"

func main() {
	payload := []byte(`{
  "recordId": "123e4567-e89b-12d3-a456-426614174000",
  "patientId": "HOSP12345",
  "patientDID": "did:example:123456abcdef",
  "providerID": "PROV123",
  "schemaVersion": "2.0",
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

	err := validation.ValidateMedicalPayload(payload)
	if err != nil {
		// fmt.Println("❌ Validation failed:", err) // Disabled for now
		os.Exit(1)
	}
	// fmt.Println(string(yellow), "✅ Medical record submission successful!", string(reset)) // Disabled for now
}