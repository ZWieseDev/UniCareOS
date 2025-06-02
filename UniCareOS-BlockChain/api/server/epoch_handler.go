package server

import (
	"encoding/json"
	"net/http"

	"unicareos/core/block"
)

// EpochStatusResponse defines the JSON structure for /epochs/status and /epochs/latest
// (latest is just an alias for status in this implementation)
//
// For Merkle root and finalized event info, see epoch_event.go and /epochs/{N} endpoint.
type EpochStatusResponse struct {
	CurrentEpoch           uint64 `json:"current_epoch"`
	BlocksInEpoch          uint64 `json:"blocks_in_epoch"`
	TotalBlocksPerEpoch    int    `json:"epoch_block_count"`
	BlocksUntilNextEpoch   int    `json:"blocks_until_next_epoch"`
}

// HandleEpochStatus returns the current epoch status and blocks until next epoch
func (s *Server) HandleEpochStatus(w http.ResponseWriter, r *http.Request) {
	state := s.network.ChainState
	if state == nil {
		http.Error(w, "Chain state unavailable", http.StatusInternalServerError)
		return
	}
	// --- Always fetch tip block from storage for live epoch/block count ---
	tipID := s.network.GetLatestBlockID()
	var blocksInEpoch uint64
	var currentEpoch uint64
	blockBytes, err := s.store.GetBlock(tipID[:])
	if err == nil && blockBytes != nil {
		tipBlock, err := block.Deserialize(blockBytes)
		if err == nil && tipBlock != nil {
			blocksInEpoch = tipBlock.Height % uint64(s.network.EpochBlockCount)
			currentEpoch = tipBlock.Epoch
		}
	}
	blocksLeft := s.network.EpochBlockCount - int(blocksInEpoch)
	if blocksLeft < 0 {
		blocksLeft = 0
	}
	resp := EpochStatusResponse{
		CurrentEpoch:         currentEpoch,
		BlocksInEpoch:        blocksInEpoch,
		TotalBlocksPerEpoch:  s.network.EpochBlockCount,
		BlocksUntilNextEpoch: blocksLeft,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Alias for /epochs/latest
func (s *Server) HandleEpochLatest(w http.ResponseWriter, r *http.Request) {
	s.HandleEpochStatus(w, r)
}
