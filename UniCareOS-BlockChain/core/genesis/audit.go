package genesis

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"crypto/sha256"
)

type AuditEvent struct {
	Timestamp time.Time       `json:"timestamp"`
	EventType string          `json:"eventType"`
	Details   json.RawMessage `json:"details"`
}

// AppendAuditEvent writes an audit event to the audit log file.
func AppendAuditEvent(event AuditEvent) error {
	f, err := os.OpenFile("genesis_audit.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

// ComputeAuditLogMerkleRoot computes a simple Merkle root of the audit log entries (hashes JSON lines).
func ComputeAuditLogMerkleRoot() (string, error) {
	f, err := os.Open("genesis_audit.log")
	if err != nil {
		return "", err
	}
	defer f.Close()
	var hashes [][]byte
	buf := make([]byte, 4096)
	var line []byte
	for {
		n, err := f.Read(buf)
		if n > 0 {
			line = append(line, buf[:n]...)
			for {
				idx := indexOf(line, '\n')
				if idx == -1 {
					break
				}
				hash := sha256Sum(line[:idx])
				hashes = append(hashes, hash)
				line = line[idx+1:]
			}
		}
		if err != nil {
			break
		}
	}
	if len(hashes) == 0 {
		return "", nil
	}
	root := merkleRoot(hashes)
	return fmt.Sprintf("%x", root), nil
}

// indexOf returns the index of sep in data, or -1 if not found.
func indexOf(data []byte, sep byte) int {
	for i, b := range data {
		if b == sep {
			return i
		}
	}
	return -1
}

// sha256Sum returns the SHA-256 hash of data.
func sha256Sum(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// mustMarshalJSON marshals v to JSON or panics if it fails (for audit logging convenience).
func mustMarshalJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mustMarshalJSON failed: %v", err))
	}
	return b
}

// merkleRoot computes the Merkle root of the given hashes.
func merkleRoot(hashes [][]byte) []byte {
	if len(hashes) == 1 {
		return hashes[0]
	}
	var nextLevel [][]byte
	for i := 0; i < len(hashes); i += 2 {
		if i+1 < len(hashes) {
			combined := append(hashes[i], hashes[i+1]...)
			nextLevel = append(nextLevel, sha256Sum(combined))
		} else {
			nextLevel = append(nextLevel, hashes[i])
		}
	}
	return merkleRoot(nextLevel)
}
