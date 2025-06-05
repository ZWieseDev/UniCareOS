package types

import (
	"time"
	"unicareos/types/ids"
)

// Block is the basic block structure shared by storage and block logic.
type Block struct {
	BlockID         ids.ID         `json:"block_id,omitempty"`
	Version         string         `json:"version"`
	ProtocolVersion string         `json:"protocolVersion"`
	Height          uint64         `json:"height"`
	PrevHash        string         `json:"prevHash"`
	MerkleRoot      string         `json:"merkleRoot"`
	Timestamp       time.Time      `json:"timestamp"`
	ValidatorDID    string         `json:"validatorDID"`
	OpUnitsUsed     uint64         `json:"opUnitsUsed"`
	Events          []Event        `json:"events"`
	BanEvents       []BanEvent     `json:"banEvents,omitempty"`
	ExtraData       []byte         `json:"extraData,omitempty"`
	ParentGasUsed   uint64         `json:"parentGasUsed,omitempty"`
	StateRoot       string         `json:"stateRoot,omitempty"`
	Signature       []byte         `json:"signature,omitempty"`
	Epoch           uint64         `json:"epoch"`
}

type Event struct {
	RecordID string `json:"recordId,omitempty"`
	EventID         ids.ID     `json:"eventID"`
	EventType       string     `json:"eventType,omitempty"`
	Description     string     `json:"description,omitempty"`
	Timestamp       time.Time  `json:"timestamp"`
	AuthorValidator ids.ID     `json:"authorValidator,omitempty"`
	Memories        []interface{} `json:"memories,omitempty"`
	PatientID       string     `json:"patientId,omitempty"`
	ProviderID      string     `json:"providerId,omitempty"`
	Epoch           uint64     `json:"epoch,omitempty"`
	PayloadHash     string     `json:"payloadHash,omitempty"`
	PayloadRef      string     `json:"payloadRef,omitempty"`
	RevisionReason  string     `json:"revisionReason,omitempty"`
	RevisionOf      string     `json:"revisionOf,omitempty"`
	DocLineage      []string   `json:"docLineage,omitempty"`
	Finalized       bool       `json:"finalized,omitempty"`
}

type BanEvent struct {
	Address   string    `json:"address"`
	Expiry    string    `json:"expiry"`
	Reason    string    `json:"reason"`
	Origin    string    `json:"origin"`
	BanCount  int       `json:"ban_count"`
	Timestamp time.Time `json:"timestamp"`
}
