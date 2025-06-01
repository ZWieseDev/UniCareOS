package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"
)

const (
	PrivKeyFile = "node_ed25519.priv"
	PubKeyFile  = "node_ed25519.pub"
)

// GenerateAndSaveKeypair generates an Ed25519 keypair and saves to disk if not present.
func GenerateAndSaveKeypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	if _, err := os.Stat(PrivKeyFile); err == nil {
		// Load existing keys
		privHex, _ := ioutil.ReadFile(PrivKeyFile)
		pubHex, _ := ioutil.ReadFile(PubKeyFile)
		priv, _ := hex.DecodeString(string(privHex))
		pub, _ := hex.DecodeString(string(pubHex))
		return ed25519.PublicKey(pub), ed25519.PrivateKey(priv), nil
	}
	// Generate new keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	ioutil.WriteFile(PrivKeyFile, []byte(hex.EncodeToString(priv)), 0600)
	ioutil.WriteFile(PubKeyFile, []byte(hex.EncodeToString(pub)), 0644)
	return pub, priv, nil
}

// LoadKeypair loads the Ed25519 keypair from disk.
func LoadKeypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	privHex, err := ioutil.ReadFile(PrivKeyFile)
	if err != nil {
		return nil, nil, err
	}
	pubHex, err := ioutil.ReadFile(PubKeyFile)
	if err != nil {
		return nil, nil, err
	}
	priv, _ := hex.DecodeString(string(privHex))
	pub, _ := hex.DecodeString(string(pubHex))
	return ed25519.PublicKey(pub), ed25519.PrivateKey(priv), nil
}

// Sign signs the message with the node's private key
func Sign(priv ed25519.PrivateKey, msg []byte) []byte {
	return ed25519.Sign(priv, msg)
}

// Verify verifies the signature with the given public key
func Verify(pub ed25519.PublicKey, msg, sig []byte) bool {
	return ed25519.Verify(pub, msg, sig)
}
