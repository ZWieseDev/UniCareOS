package blockchain

import (
	"encoding/json"
	"sort"
	"unicareos/core/block"
	"unicareos/core/storage"
)

// EventHashEntry holds event hash with its block and index for deterministic ordering
type EventHashEntry struct {
	BlockHeight int
	EventIndex  int
	Hash        string
}

// GatherFinalizedEventHashesForEpoch returns a deterministically ordered list of finalized event hashes for a given epoch.
func GatherFinalizedEventHashesForEpoch(epoch uint64, store *storage.Storage) ([]string, error) {
	var entries []EventHashEntry
	blockIDs, err := store.ListBlockIDs()
	if err != nil {
		return nil, err
	}
	for _, blockID := range blockIDs {
		blockBytes, err := store.GetBlock(blockID)
		if err != nil { continue }
		var blk block.Block
		if err := json.Unmarshal(blockBytes, &blk); err != nil { continue }
		if blk.Epoch != epoch { continue }
		for idx, evt := range blk.Events {
			if evt.EventType == "finalize_event" {
				// Convert evt to FinalizeEventTx if needed (assumes evt is compatible)
				tx := block.FinalizeEventTx{}
				// If evt is already a FinalizeEventTx, use directly; else, map fields as needed
				b, _ := json.Marshal(evt)
				if err := json.Unmarshal(b, &tx); err != nil { continue }
				hash := block.HashFinalizeEventTx(&tx)
				entries = append(entries, EventHashEntry{int(blk.Height), idx, hash})
			}
		}
	}
	// Sort by block height, then event index
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].BlockHeight != entries[j].BlockHeight {
			return entries[i].BlockHeight < entries[j].BlockHeight
		}
		return entries[i].EventIndex < entries[j].EventIndex
	})
	hashes := make([]string, len(entries))
	for i, entry := range entries {
		hashes[i] = entry.Hash
	}
	return hashes, nil
}

// ComputeEpochMerkleRoot returns the Merkle root for all finalized events in the given epoch.
func ComputeEpochMerkleRoot(epoch uint64, store *storage.Storage) (string, error) {
	hashes, err := GatherFinalizedEventHashesForEpoch(epoch, store)
	if err != nil {
		return "", err
	}
	return block.MerkleRoot(hashes), nil
}
