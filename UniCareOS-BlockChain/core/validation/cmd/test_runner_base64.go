package main

import (
	"fmt"
	"os"
	"unicareos/core/validation"
)

func main() {

	payload := []byte(`{
  "recordId": "123e4567-e89b-12d3-a456-426614174000",
  "patientId": "NOT-BASE64!!",
  "patientDID": "ZGlkOmV4YW1wbGU6MTIzNDU2YWJjZGVm", 
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
  "payloadSignature": "YWJjZGVmZ2hpamtsbW5vcA=="
}`)

	err := validation.ValidateMedicalPayload(payload)
	if err != nil {
		// fmt.Println("❌ Base64 test failed as expected:", err) // Disabled for now
		os.Exit(1)
	}
	// fmt.Println("❌ Base64 test should have failed, but passed!") // Disabled for now
	os.Exit(2)
}
