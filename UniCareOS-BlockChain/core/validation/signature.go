package validation

import (
	"fmt"
	"encoding/json"
	"os"
	"io/ioutil"
	"encoding/base64"
	"unicareos/core"
)

// VerifyWalletSignature verifies that the payload was signed by the holder of the given wallet address.
// Returns nil if the signature is valid, or an error if invalid or verification fails.
func VerifyWalletSignature(payload map[string]interface{}, signature string, walletAddress string) error {
	// 1. Load authorized_wallets.json
	file, err := os.Open("core/block/authorized_wallets.json")
	if err != nil {
		return fmt.Errorf("failed to open authorized_wallets.json: %v", err)
	}
	defer file.Close()
	var wallets map[string]struct {
		Authorized bool   `json:"authorized"`
		PublicKey  string `json:"publicKey"`
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read authorized_wallets.json: %v", err)
	}
	if err := json.Unmarshal(bytes, &wallets); err != nil {
		return fmt.Errorf("failed to parse authorized_wallets.json: %v", err)
	}

	// Log the loaded wallets for debugging (REDACTED public keys)
	// fmt.Println("[Wallet Allowlist] Loaded wallets:")
	// for addr, info := range wallets {
	// 	fmt.Printf("  - %s: authorized=%v\n", addr, info.Authorized)
	// }
	// fmt.Printf("[DEBUG] Incoming walletAddress: '%s'\n", walletAddress)
	// fmt.Println("[DEBUG] Loaded allowlist keys:")
	// for k := range wallets {
	// 	fmt.Printf("  - '%s'\n", k)
	// }
	w, ok := wallets[walletAddress]
	// fmt.Printf("[DEBUG] Lookup result: found=%v, authorized=%v\n", ok, w.Authorized)
	if !ok || !w.Authorized {
		return fmt.Errorf("wallet address not authorized")
	}
	if w.PublicKey == "" {
		return fmt.Errorf("no public key for wallet address")
	}

	// 2. Serialize payload deterministically (canonical JSON)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload: %v", err)
	}

	// 3. (No need to decode signature here; handled by core.VerifySignature or downstream logic)

	// 4. Decode public key (assume base64 for now)
	// --- Signature Verification Debug Section (REDACTED) ---
	pubKeyBytes, err := base64.StdEncoding.DecodeString(w.PublicKey)
	if err != nil {
		fmt.Printf("[ERROR] Public key base64 decode failed: %v\n", err)
		return fmt.Errorf("invalid public key encoding: %v", err)
	}
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		fmt.Printf("[ERROR] Signature base64 decode failed: %v\n", err)
		return fmt.Errorf("invalid signature encoding: %v", err)
	}
	if len(sigBytes) != 64 {
		fmt.Printf("[WARN] Signature length is not 64 bytes (Ed25519 signature)!\n")
	}
	if len(pubKeyBytes) != 32 {
		fmt.Printf("[WARN] Public key length is not 32 bytes (Ed25519 public key)!\n")
	}
	// Print only SHA-256 hash of payload for troubleshooting
	//hash := sha256.Sum256(payloadBytes)
	//fmt.Printf("[DEBUG] SHA-256(payloadBytes): %x\n", hash)
	// No raw payload, signature, or public key bytes are logged

	// 5. Construct core.Signature object (minimal fields)
	sig := core.Signature{
		Algorithm: "Ed25519", // or "ECDSA" if you support others
		Signature: signature,
		SignerAddress: walletAddress,
	}

	// 6. Call core.VerifySignature
	if !core.VerifySignature(sig, pubKeyBytes, payloadBytes) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}
