package block

import (
	"crypto/sha256"
	"encoding/hex"
)

// MerkleRoot computes the Merkle root of a list of hashes (as hex strings).
// If the list is empty, returns an empty string.
func MerkleRoot(hashes []string) string {
	n := len(hashes)
	if n == 0 {
		return ""
	}
	for n > 1 {
		var nextLevel []string
		for i := 0; i < n; i += 2 {
			if i+1 < n {
				h := sha256.New()
				h.Write([]byte(hashes[i]))
				h.Write([]byte(hashes[i+1]))
				nextLevel = append(nextLevel, hex.EncodeToString(h.Sum(nil)))
			} else {
				// Odd node: hash with itself
				h := sha256.New()
				h.Write([]byte(hashes[i]))
				h.Write([]byte(hashes[i]))
				nextLevel = append(nextLevel, hex.EncodeToString(h.Sum(nil)))
			}
		}
		hashes = nextLevel
		n = len(hashes)
	}
	return hashes[0]
}

// HashFinalizeEventTx returns the SHA-256 hash (hex) of the canonical JSON of the FinalizeEventTx
func HashFinalizeEventTx(tx *FinalizeEventTx) string {
	data, _ := tx.MarshalCanonical()
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
