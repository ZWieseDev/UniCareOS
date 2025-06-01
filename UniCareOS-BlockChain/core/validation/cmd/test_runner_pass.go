package main

import (
	"fmt"
	"os"
	"unicareos/core/validation"
)

func main() {

	payload := []byte(`{
  "recordId": "123e4567-e89b-12d3-a456-426614174000",
  "patientId": "UFJPVjEyMzQ1", 
  "patientDID": "did:example:123456abcdef",
  "providerID": "UFJPVjEyMw==",
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
  "payloadSignature": "YWJjZGVmZ2hpamtsbW5vcA==",
  "notes": "U29tZSB2YWxpZCBub3RlcyBoZXJlIg=="
}`)

	err := validation.ValidateMedicalPayload(payload)
	if err != nil {
		// fmt.Println("❌ PASS test failed (should have passed):", err) // Disabled for now
		os.Exit(1)
	}
	// fmt.Println("✅ PASS test succeeded: payload is valid!") // Disabled for now
}
