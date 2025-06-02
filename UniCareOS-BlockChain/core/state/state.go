package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"unicareos/core/block"
	"unicareos/core/storage"
)

// Duplicate ChainState definition commented out to avoid redeclaration errors.
// type ChainState struct {
// 	// ... fields as before ...
// }

// LoadEpochState loads epoch-related fields from StateDB.
func (cs *ChainState) LoadEpochState() error {
	if cs.StateDB == nil {
		return errors.New("StateDB is nil")
	}
	// Load Epoch
	if data, err := cs.StateDB.Get("current_epoch"); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &cs.Epoch); err != nil {
			return err
		}
	}
	// Load BlocksInEpoch
	if data, err := cs.StateDB.Get("blocks_in_epoch"); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &cs.BlocksInEpoch); err != nil {
			return err
		}
	}
	return nil
}

// SaveEpochState persists epoch-related fields to StateDB.
func (cs *ChainState) SaveEpochState() error {
	if cs.StateDB == nil {
		return errors.New("StateDB is nil")
	}
	// Save Epoch
	epochBytes, err := json.Marshal(cs.Epoch)
	if err != nil {
		return err
	}
	if err := cs.StateDB.Put("current_epoch", epochBytes); err != nil {
		return err
	}
	// Save BlocksInEpoch
	blocksBytes, err := json.Marshal(cs.BlocksInEpoch)
	if err != nil {
		return err
	}
	if err := cs.StateDB.Put("blocks_in_epoch", blocksBytes); err != nil {
		return err
	}
	return nil
}

// ChainState represents the persistent blockchain state.
type ChainState struct {
	ChainHead     string
	Height        uint64
	Epoch         uint64
	BlocksInEpoch uint64 // Number of blocks committed in the current epoch
	StateDB       storage.StateBackend
	Indexes       ChainIndexes
}

type ChainIndexes struct {
	ByHash       map[string]uint64
	ByPatientID  map[string][]string
	ByProviderID map[string][]string
	ByEpoch      map[uint64][]string
}

type StateUpdateReceipt struct {
	BlockHash string   `json:"blockHash"`
	Status    string   `json:"status"`
	Timestamp string   `json:"timestamp"`
	Errors    []string `json:"errors"`
}

type AuditLogEntry struct {
	BlockHash string `json:"blockHash"`
	Timestamp string `json:"timestamp"`
	UpdatedBy string `json:"updatedBy"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
}

// WriteBlockToState persists a finalized block and updates chain state.
func WriteBlockToState(state *ChainState, blk *block.Block, updatedBy string) (StateUpdateReceipt, error) {
	receipt := StateUpdateReceipt{
		BlockHash: blk.BlockID.String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if blk == nil {
		receipt.Status = "failed"
		receipt.Errors = append(receipt.Errors, "nil_block")
		LogStateUpdate(blk.BlockID.String(), receipt.Status, "Block is nil")
		return receipt, errors.New("block is nil")
	}
	// Validate MerkleRoot, blockID, etc. (TODO: add real validation)
	if blk.MerkleRoot == "" || blk.BlockID.String() == "" {
		receipt.Status = "failed"
		receipt.Errors = append(receipt.Errors, "invalid_block")
		LogStateUpdate(blk.BlockID.String(), receipt.Status, "Invalid block fields")
		return receipt, errors.New("invalid block fields")
	}
	// Persist block to state DB
	blockKey := fmt.Sprintf("block:%s", blk.BlockID.String())
	blockBytes, err := json.Marshal(blk)
	if err != nil {
		receipt.Status = "failed"
		receipt.Errors = append(receipt.Errors, "marshal_error")
		LogStateUpdate(blk.BlockID.String(), receipt.Status, "Marshal error")
		return receipt, err
	}
	if err := state.StateDB.Put(blockKey, blockBytes); err != nil {
		receipt.Status = "failed"
		receipt.Errors = append(receipt.Errors, "db_write_error")
		LogStateUpdate(blk.BlockID.String(), receipt.Status, "DB write error")
		return receipt, err
	}
	// Update chain head and indexes
	SetChainHead(state, blk)
	IndexEventsByPatientAndType(state, blk.Events)
	receipt.Status = "committed"
	LogStateUpdate(blk.BlockID.String(), receipt.Status, "Block committed")
	return receipt, nil
}

// SetChainHead updates the canonical chain head pointer.
func SetChainHead(state *ChainState, blk *block.Block) {
	state.ChainHead = blk.BlockID.String()
	state.Height = blk.Height
	state.Epoch = blk.Epoch
	// Optionally persist chain head pointer
	_ = state.StateDB.Put("chain_head", []byte(blk.BlockID.String()))
}

// IndexEventsByPatientAndType indexes block events for fast lookup.
func IndexEventsByPatientAndType(state *ChainState, events []block.ChainedEvent) {
	for _, evt := range events {
		pid := evt.PatientID
		provider := evt.ProviderID
		if pid != "" {
			state.Indexes.ByPatientID[pid] = append(state.Indexes.ByPatientID[pid], evt.EventID.String())
		}
		if provider != "" {
			state.Indexes.ByProviderID[provider] = append(state.Indexes.ByProviderID[provider], evt.EventID.String())
		}
		state.Indexes.ByEpoch[evt.Epoch] = append(state.Indexes.ByEpoch[evt.Epoch], evt.EventID.String())
	}
}

// GetBlockMetadata retrieves metadata for a block by hash.
func GetBlockMetadata(state *ChainState, blockHash string) (*block.Block, error) {
	blockKey := fmt.Sprintf("block:%s", blockHash)
	data, err := state.StateDB.Get(blockKey)
	if err != nil {
		return nil, err
	}
	var blk block.Block
	if err := json.Unmarshal(data, &blk); err != nil {
		return nil, err
	}
	return &blk, nil
}

// LogStateUpdate logs a state update for audit/compliance.
func LogStateUpdate(blockHash, status, reason string) {
	entry := AuditLogEntry{
		BlockHash: blockHash,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		UpdatedBy: uuid.New().String(), // TODO: Use real user
		Status:    status,
		Reason:    reason,
	}
	// TODO: Persist to audit log (file, db, etc.)
	fmt.Printf("[AUDIT] %+v\n", entry)
}
