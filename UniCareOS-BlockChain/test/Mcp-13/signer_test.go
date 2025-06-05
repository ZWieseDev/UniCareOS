package Mcp13

import (

	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"testing"


	"github.com/stretchr/testify/require"

	"unicareos/core"
)

func TestEd25519SignAndVerify(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	wallet := core.Wallet{
		Address:    "0xEDTEST",
		PublicKey:  pub,
		PrivateKey: priv,
		Algorithm:  "Ed25519",
	}
	payload := []byte("test payload ed25519")

	sig, err := core.SignTransaction(wallet, payload)
	require.NoError(t, err)

	ok := core.VerifySignature(sig, pub, payload)
	require.True(t, ok, "Ed25519 signature should verify")
}

func TestECDSASignAndVerify(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Encode private key to DER
	der, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	// Encode public key to DER
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)

	wallet := core.Wallet{
		Address:    "0xECDSATest",
		PublicKey:  pubBytes,
		PrivateKey: der,
		Algorithm:  "ECDSA",
	}
	payload := []byte("test payload ecdsa")

	sig, err := core.SignTransaction(wallet, payload)
	require.NoError(t, err)

	ok := core.VerifySignature(sig, pubBytes, payload)
	require.True(t, ok, "ECDSA signature should verify")
}

func TestSignatureTampering(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	wallet := core.Wallet{
		Address:    "0xEDTEST2",
		PublicKey:  pub,
		PrivateKey: priv,
		Algorithm:  "Ed25519",
	}
	payload := []byte("test payload ed25519 tamper")

	sig, err := core.SignTransaction(wallet, payload)
	require.NoError(t, err)

	// Tamper with payload
	ok := core.VerifySignature(sig, pub, []byte("tampered payload"))
	require.False(t, ok, "Signature should not verify for tampered payload")
}
