package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"unicareos/core/wallet"
)

func main() {
	// Load private key from environment variable using EnvWalletLoader
	loader := wallet.EnvWalletLoader{}
	w, err := loader.LoadWallet()
	if err != nil {
		panic(err)
	}
	privKeyB64 := w.PrivateKey
	privKey, err := base64.StdEncoding.DecodeString(privKeyB64)
	if err != nil {
		panic(err)
	}
	// Print the derived public key for debug
	if len(privKey) == 64 {
		pubKey := privKey[32:]
		fmt.Fprintf(os.Stderr, "Derived public key: %s\n", base64.StdEncoding.EncodeToString(pubKey))
	} else {
		fmt.Fprintf(os.Stderr, "WARNING: Private key length is not 64 bytes, cannot derive public key!\n")
	}

	record := map[string]interface{}{
		"recordId": "b0f3e7a4-1e8b-4c0e-bc8d-7c9b6a6c2e4f",
		"patientId": "YWJjZGVmZ2hpamtsbW5vcA==",
		"patientDID": "did:example:123456789abcdefghi",
		"providerId": "encrypted-provider-id",
		"docHash": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"schemaVersion": "1.0",
		"recordType": "lab_result",
		"issuedAt": "2025-05-25T15:00:00Z",
		"signedBy": "test_wallet",
		"retentionPolicy": "standard",
		"encryptionContext": map[string]interface{}{
			"algorithm": "AES-GCM",
			"iv": "abcdefghijklmnop1234",
			"tag": "ZYXWVUTSRQPONMLK9876",
		},
		"consentStatus": "granted",
		"dataProvenance": "hospital-system",
	}

	// Marshal record for signing
	payloadBytes, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "RECORD JSON TO SIGN: %s\n", string(payloadBytes))
	hash := sha256.Sum256(payloadBytes)
	fmt.Fprintf(os.Stderr, "SHA-256(record): %x\n", hash)

	sig := ed25519.Sign(privKey, hash[:])
	sigB64 := base64.StdEncoding.EncodeToString(sig)
	fmt.Fprintf(os.Stderr, "SIGNATURE (base64): %s\n", sigB64)

	// Now add the payloadSignature field
	record["payloadSignature"] = sigB64

	// Marshal record for signing
	payloadBytes, err = json.Marshal(record)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "RECORD JSON TO SIGN: %s\n", string(payloadBytes))
	hash = sha256.Sum256(payloadBytes)
	fmt.Fprintf(os.Stderr, "SHA-256(record): %x\n", hash)

	sig = ed25519.Sign(privKey, hash[:])
	sigB64 = base64.StdEncoding.EncodeToString(sig)
	fmt.Fprintf(os.Stderr, "SIGNATURE (base64): %s\n", sigB64)

	finalPayload := map[string]interface{}{
		"record": record,
		"signature": sigB64,
		"walletAddress": "test_wallet",
	}

	finalJSON, err := json.MarshalIndent(finalPayload, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(finalJSON))

	if len(os.Args) > 1 {
		err = os.WriteFile(os.Args[1], finalJSON, 0644)
		if err != nil {
			panic(err)
		}
	}
}
