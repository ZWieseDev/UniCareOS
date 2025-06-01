package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

// getDEK retrieves the Data Encryption Key from the environment (base64-encoded, 32 bytes after decoding)
func getDEK() ([]byte, error) {
	dekB64 := os.Getenv("UNICARE_DEK")
	if dekB64 == "" {
		return nil, errors.New("UNICARE_DEK not set in environment")
	}
	dek, err := base64.StdEncoding.DecodeString(dekB64)
	if err != nil {
		return nil, errors.New("failed to decode UNICARE_DEK: " + err.Error())
	}
	if len(dek) != 32 {
		return nil, errors.New("UNICARE_DEK must be 32 bytes (base64-encoded)")
	}
	return dek, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and a random nonce
func Encrypt(plaintext []byte) ([]byte, error) {
	dek, err := getDEK()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func Decrypt(ciphertext []byte) ([]byte, error) {
	dek, err := getDEK()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}
