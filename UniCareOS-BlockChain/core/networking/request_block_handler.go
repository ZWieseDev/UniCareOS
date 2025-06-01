package networking

import (
	"encoding/hex"
	"net/http"
	"unicareos/core/storage"
)

// RequestBlockHandler serves block bytes for a given block ID as a HTTP endpoint
func RequestBlockHandler(store *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		blockIDHex := r.URL.Query().Get("block_id")
		if blockIDHex == "" {
			http.Error(w, "missing block_id", http.StatusBadRequest)
			return
		}
		blockID, err := hex.DecodeString(blockIDHex)
		if err != nil || len(blockID) != 32 {
			http.Error(w, "invalid block_id", http.StatusBadRequest)
			return
		}
		blkBytes, err := store.GetBlock(blockID)
		if err != nil {
			http.Error(w, "block not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(blkBytes)
	}
}
