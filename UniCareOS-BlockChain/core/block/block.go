package block

import (
	"encoding/json"
	"time"
	"unicareos/types/ids"
)

// BanEvent now lives in types/blocktypes.go
type BanEvent struct {
	Address   string    `json:"address"`   // Banned peer IP or identifier
	Expiry    string    `json:"expiry"`    // RFC3339 expiry time
	Reason    string    `json:"reason"`    // Optional reason/evidence
	Origin    string    `json:"origin"`    // Node ID or validator
	BanCount  int       `json:"ban_count"` // Number of bans for this address
	Timestamp time.Time `json:"timestamp"` // When the ban was issued
}

// ExpiryTime parses the Expiry string and returns it as time.Time
func (b BanEvent) ExpiryTime() (time.Time, error) {
	return time.Parse(time.RFC3339, b.Expiry)
}


type Block struct {
	BlockID         ids.ID         `json:"block_id,omitempty"`      // Computed or cached block hash
	Version         string         `json:"version"`          // Spec version of block structure
	ProtocolVersion string         `json:"protocolVersion"`  // Protocol release tag
	Height          uint64         `json:"height"`           // Block height (genesis = 0)
	PrevHash        string         `json:"prevHash"`         // Parent block hash
	MerkleRoot      string         `json:"merkleRoot"`       // Merkle root of transactions/events
	Timestamp       time.Time      `json:"timestamp"`        // ISO-8601 UTC timestamp
	ValidatorDID    string         `json:"validatorDID"`     // DID of block producer
	OpUnitsUsed     uint64         `json:"opUnitsUsed"`      // Optional: operation units consumed
	Events          []ChainedEvent `json:"events"`           // Block events/transactions
	AuditLog        []AuditLogEntry `json:"auditLog,omitempty"` // Medical record submission audit log
	BanEvents       []BanEvent     `json:"banEvents,omitempty"` // ✅ Ban events included in block
	ExtraData       []byte         `json:"extraData,omitempty"`     // Reserved for future protocol flags (32 bytes)
	ParentGasUsed   uint64         `json:"parentGasUsed,omitempty"` // For gas metrics (future)
	StateRoot       string         `json:"stateRoot,omitempty"`     // Global state snapshot (future)
	Signature       []byte         `json:"signature,omitempty"`     // Block producer's digital signature
	Epoch           uint64         `json:"epoch"`                   // Epoch number
	
}

// ComputeID computes the hash of the block header fields (excluding BlockID itself)
func (b *Block) ComputeID() ids.ID {
	header := struct {
		Version         string
		ProtocolVersion string
		Height          uint64
		PrevHash        string
		MerkleRoot      string
		Timestamp       time.Time
		ValidatorDID    string
		OpUnitsUsed     uint64
		ExtraData       []byte
		ParentGasUsed   uint64
		StateRoot       string
		Epoch           uint64
	}{
		b.Version, b.ProtocolVersion, b.Height, b.PrevHash, b.MerkleRoot,
		b.Timestamp, b.ValidatorDID, b.OpUnitsUsed, b.ExtraData, b.ParentGasUsed, b.StateRoot,
		b.Epoch,
	}
	data, _ := json.Marshal(header)
	return ids.NewID(data)
}

// Serialize encodes Block into JSON
func (b *Block) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

// Deserialize decodes JSON into Block
func Deserialize(data []byte) (*Block, error) {
	var b Block
	err := json.Unmarshal(data, &b)
	if err != nil {
		return nil, err
	}
	return &b, nil
}