package blockchain

import (
	"fmt"
	"time"
	"unicareos/core/storage"
	"unicareos/core/state"
	"unicareos/core/types"
)

// FinalizeEpoch seals an epoch, computes its Merkle root, and creates a FinalizeEpochTx.
func FinalizeEpoch(
	store *storage.Storage,
	chainState *state.ChainState,
	epochNumber uint64,
	finalizerSignature string,
	auditLogID string,
) (*types.FinalizeEpochTx, *types.EpochFinalizationReceipt, error) {
	// Check if the epoch is already finalized (TODO: implement actual check if needed)
	// For now, assume single finalization per epoch

	// Gather finalized event hashes and compute Merkle root
	_, err := GatherFinalizedEventHashesForEpoch(epochNumber, store)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to gather event hashes: %w", err)
	}
	root, err := ComputeEpochMerkleRoot(epochNumber, store)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute epoch Merkle root: %w", err)
	}

	tx := types.NewFinalizeEpochTx(epochNumber, finalizerSignature, root)
	tx.AuditLogID = auditLogID
	tx.Timestamp = time.Now().UTC()

	if err := tx.Validate(); err != nil {
		tx.Status = "failed"
		receipt := &types.EpochFinalizationReceipt{
			TxID:        tx.TxID,
			Status:      tx.Status,
			EpochNumber: epochNumber,
			Timestamp:   tx.Timestamp,
			Errors:      []string{err.Error()},
		}
		return tx, receipt, err
	}

	// Mark epoch as finalized in chain state if needed (TODO: implement marking logic)
	// chainState.MarkEpochFinalized(epochNumber) // example placeholder

	tx.Status = "finalized"
	receipt := &types.EpochFinalizationReceipt{
		TxID:        tx.TxID,
		Status:      tx.Status,
		EpochNumber: epochNumber,
		Timestamp:   tx.Timestamp,
	}
	// TODO: Persist FinalizeEpochTx to storage or chain if required

	return tx, receipt, nil
}
