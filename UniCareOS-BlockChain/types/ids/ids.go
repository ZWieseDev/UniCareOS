package ids

import (
	"crypto/sha256"
	"encoding/hex"
)

// ID is a 32-byte array.
type ID [32]byte

// Empty is the zero-value ID (all zeros)
var Empty ID

// NewID generates a new ID by hashing input bytes
func NewID(data []byte) ID {
	hash := sha256.Sum256(data)
	return ID(hash)
}

// FromString parses a hex string into an ID
func FromString(s string) (ID, error) {
	var id ID
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return id, err
	}
	copy(id[:], bytes)
	return id, nil
}

// String converts an ID back to a hex string
func (id ID) String() string {
	return hex.EncodeToString(id[:])
}

// IDFromString creates an ID from a string (using SHA-256)
func IDFromString(s string) ID {
	return NewID([]byte(s))
}
