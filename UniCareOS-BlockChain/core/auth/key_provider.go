package auth

import (
	"crypto/rsa"
	"errors"
)

type KeyProvider interface {
	GetPublicKey(kid string) (interface{}, error)
}

// DummyKeyProvider loads a hardcoded or env-based RSA public key for dev/testing.
type DummyKeyProvider struct {
	PublicKey *rsa.PublicKey
}

func (d *DummyKeyProvider) GetPublicKey(kid string) (interface{}, error) {
	if d.PublicKey != nil {
		return d.PublicKey, nil
	}
	return nil, errors.New("no public key set")
}
