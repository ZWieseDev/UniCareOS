package blockchain

import (
	"time"
	"unicareos/core/block"
	"unicareos/core/state"
	"unicareos/types/ids"
)

// WriteFinalizedEventToChain writes a finalized event (with S3 URL/hash) to the blockchain.
func WriteFinalizedEventToChain(
	st *state.ChainState,
	blockID ids.ID,
	patientID string,        // should be encrypted before passing
	providerID string,
	epoch uint64,
	payloadHash string,      // hash of the encrypted file
	payloadRef string,       // S3 URL or secure pointer
	description string,
	updatedBy string,
) error {
	event := block.ChainedEvent{
		EventID:      ids.ID{}, // TODO: generate or pass a unique event ID
		EventType:    "finalize_event",
		Description:  description,
		Timestamp:    time.Now().UTC(),
		PatientID:    patientID,
		ProviderID:   providerID,
		Epoch:        epoch,
		PayloadHash:  payloadHash,
		PayloadRef:   payloadRef,
	}

	blk := &block.Block{
		BlockID:    blockID,
		MerkleRoot: "", // can be set after events are finalized
		Height:     1,  // set appropriately
		Epoch:      epoch,
		Events:     []block.ChainedEvent{event},
	}

	receipt, err := state.WriteBlockToState(st, blk, updatedBy)
	if err != nil || receipt.Status != "committed" {
		return err
	}
	return nil
}
