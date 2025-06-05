package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"unicareos/core/blockchain"
)

// EpochEventResponse defines the JSON structure for /epochs/{N}
type EpochEventResponse struct {
	Epoch      uint64   `json:"epoch"`
	MerkleRoot string   `json:"merkle_root"`
	EventCount int      `json:"event_count"`
	Hashes     []string `json:"hashes,omitempty"`
}

// HandleEpochEvent returns the Merkle root and event info for a given epoch
func (s *Server) HandleEpochEvent(w http.ResponseWriter, r *http.Request) {
	// Expect URL like /epochs/{N}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid epoch path", http.StatusBadRequest)
		return
	}
	epochStr := parts[2]
	epoch, err := strconv.ParseUint(epochStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid epoch number", http.StatusBadRequest)
		return
	}
	// Compute Merkle root and event hashes
	hashes, err := blockchain.GatherFinalizedEventHashesForEpoch(epoch, s.store)
	if err != nil {
		http.Error(w, "Failed to gather event hashes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	root := ""
	if len(hashes) > 0 {
		root, err = blockchain.ComputeEpochMerkleRoot(epoch, s.store)
		if err != nil {
			http.Error(w, "Failed to compute Merkle root: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	resp := EpochEventResponse{
		Epoch:      epoch,
		MerkleRoot: root,
		EventCount: len(hashes),
		Hashes:     hashes, // Remove if you want to hide hashes from API
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
