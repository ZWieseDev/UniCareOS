package core

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"
	"crypto/x509"
	"math/big"
)

// Wallet represents a signing wallet (private key should be protected)
type Wallet struct {
	Address    string
	PublicKey  []byte
	PrivateKey []byte // Never expose outside secure enclave
	Algorithm  string // "ECDSA" or "Ed25519"
}

// Signature contains all signature metadata
type Signature struct {
	Algorithm        string    `json:"algorithm"`
	Signature        string    `json:"signature"`
	SignedPayloadHash string   `json:"signedPayloadHash"`
	SignerAddress    string    `json:"signerAddress"`
	Timestamp        time.Time `json:"timestamp"`
}

// SignTransaction signs a payload and returns a Signature object
func SignTransaction(wallet Wallet, payload []byte) (Signature, error) {
	hash := sha256.Sum256(payload)
	ts := time.Now().UTC()
	var sig []byte

	switch wallet.Algorithm {
	case "Ed25519":
		if len(wallet.PrivateKey) != ed25519.PrivateKeySize {
			return Signature{}, errors.New("invalid Ed25519 private key size")
		}
		sig = ed25519.Sign(wallet.PrivateKey, hash[:])
	case "ECDSA":
		priv, err := BytesToECDSA(wallet.PrivateKey)
		if err != nil {
			return Signature{}, err
		}
		r, s, err := ecdsa.Sign(rand.Reader, priv, hash[:])
		if err != nil {
			return Signature{}, err
		}
		sig = append(r.Bytes(), s.Bytes()...)
	default:
		return Signature{}, errors.New("unsupported algorithm")
	}

	return Signature{
		Algorithm:        wallet.Algorithm,
		Signature:        base64.StdEncoding.EncodeToString(sig),
		SignedPayloadHash: hex.EncodeToString(hash[:]),
		SignerAddress:    wallet.Address,
		Timestamp:        ts,
	}, nil
}

// VerifySignature verifies a signature for a payload and public key
func VerifySignature(sig Signature, pubKey []byte, payload []byte) bool {
	hash := sha256.Sum256(payload)
	switch sig.Algorithm {
	case "Ed25519":
		return ed25519.Verify(pubKey, hash[:], decodeB64(sig.Signature))
	case "ECDSA":
		ecdsaPub, err := BytesToECDSAPub(pubKey) // DER-encoded public key
		if err != nil {
			return false
		}
		r, s := decodeECDSASig(decodeB64(sig.Signature))
		return ecdsa.Verify(ecdsaPub, hash[:], r, s)
	default:
		return false
	}
}

// IsWhitelistedSigner checks if a public key is in the authorized list
func IsWhitelistedSigner(pubKey []byte) bool {
	// TODO: implement actual whitelist logic
	return true
}

// LoadWalletFromSecretsManager is a placeholder for secure key retrieval
func LoadWalletFromSecretsManager(address string) (Wallet, error) {
	// TODO: implement secure key retrieval
	return Wallet{}, errors.New("not implemented")
}

// Helpers (implementations for key conversion and signature decoding)


func BytesToECDSA(priv []byte) (*ecdsa.PrivateKey, error) {
	// Accept PKCS#8 or raw EC private key
	key, err := x509.ParseECPrivateKey(priv)
	if err == nil {
		return key, nil
	}
	// Try PKCS#8
	k, err := x509.ParsePKCS8PrivateKey(priv)
	if err == nil {
		if ecdsaKey, ok := k.(*ecdsa.PrivateKey); ok {
			return ecdsaKey, nil
		}
	}
	return nil, errors.New("invalid ECDSA private key format")
}

func BytesToECDSAPub(pub []byte) (*ecdsa.PublicKey, error) {
	// Accept ASN.1 DER EC public key
	key, err := x509.ParsePKIXPublicKey(pub)
	if err == nil {
		if ecdsaPub, ok := key.(*ecdsa.PublicKey); ok {
			return ecdsaPub, nil
		}
	}
	return nil, errors.New("invalid ECDSA public key format")
}

func decodeB64(s string) []byte {
	b, _ := base64.StdEncoding.DecodeString(s)
	return b
}

func decodeECDSASig(sig []byte) (r, s *big.Int) {
	// Assume signature is r||s, each 32 bytes (for secp256k1)
	if len(sig) != 64 {
		return nil, nil
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:])
	return r, s
}
