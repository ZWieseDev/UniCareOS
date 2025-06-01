package wallet

import (
	"crypto/ed25519"
)

type SignatureVerifier interface {
	VerifySignature(payload []byte, signature []byte, walletAddr string) bool
}

// Ed25519Verifier implements SignatureVerifier for Ed25519 keys.
type Ed25519Verifier struct {
	// Map of walletAddr to public keys (for demo)
	PublicKeys map[string]ed25519.PublicKey
}

func (v *Ed25519Verifier) VerifySignature(payload []byte, signature []byte, walletAddr string) bool {
	pub, ok := v.PublicKeys[walletAddr]
	if !ok {
		return false
	}
	return ed25519.Verify(pub, payload, signature)
}
