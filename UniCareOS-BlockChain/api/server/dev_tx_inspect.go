//Dev delete upon production migration
// This endpoint is for development/testing only. It allows inspection of a transaction's decoded contents by txID.
package server

import (
	"encoding/json"
	"net/http"
	"strings"

)

// RegisterDevTxInspectAPI registers the dev-only transaction inspection endpoint.
func RegisterDevTxInspectAPI(mux *http.ServeMux, s *Server) {
	mux.HandleFunc("/dev/inspect_tx", s.handleDevInspectTx)
}

// handleDevInspectTx returns the decoded transaction contents for a given txID (dev only)
func (s *Server) handleDevInspectTx(w http.ResponseWriter, r *http.Request) {
	var txPayload []byte
	found := false
	if r.Method != http.MethodGet {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}
	txID := r.URL.Query().Get("txId")
	if txID == "" {
		http.Error(w, "missing txId parameter", http.StatusBadRequest)
		return
	}
	// Search mempool and blocks for the txID
	found = false
	// 1. Search mempool
	if s.gossipEngine != nil {
		tx, ok := s.gossipEngine.Mempool.GetTx(txID)
		if ok {
			txPayload = tx.Payload
			found = true
		}
	}
	// 2. Search blocks if not found in mempool
	if !found && s.store != nil {
		height, err := s.store.GetChainHeight()
		if err == nil {
			for i := 0; i < height; i++ {
				blk, err := s.store.GetBlockByHeight(i)
				if err != nil { continue }
				// Search blk.Events for event with matching EventID as txID
				for _, evt := range blk.Events {
					if strings.EqualFold(evt.EventID.String(), txID) {
						txPayload, _ = json.Marshal(evt)
						found = true
						break
					}
				}
				if found { break }
			}
		}
	}
	if found {
		if s.gossipEngine != nil {
			tx, ok := s.gossipEngine.Mempool.GetTx(txID)
			if ok {
				txPayload = tx.Payload
			}
		} else if s.store != nil {
			height, err := s.store.GetChainHeight()
			if err == nil {
				for i := 0; i < height; i++ {
					blk, err := s.store.GetBlockByHeight(i)
					if err != nil { continue }
					// Search blk.Events for event with matching EventID as txID
					for _, evt := range blk.Events {
						if strings.EqualFold(evt.EventID.String(), txID) {
							txPayload = []byte{} // clear txPayload
							txPayload, _ = json.Marshal(evt)
							break
						}
					}
					if len(txPayload) > 0 { break }
				}
			}
		}
	}
	if !found {
		http.Error(w, "tx not found", http.StatusNotFound)
		return
	}
	// Attempt to decode payload as JSON
	var payloadObj map[string]interface{}
	var medicalRecord interface{} = nil
	if err := json.Unmarshal(txPayload, &payloadObj); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"txId":    txID,
			"payload": string(txPayload),
			"note":    "Payload is not valid JSON; showing as base64 string.",
		})
		return
	}
	// Try to extract the medical record
	if rec, ok := payloadObj["Record"]; ok {
		medicalRecord = rec
	} else if rec, ok := payloadObj["record"]; ok {
		medicalRecord = rec
	} else if payload, ok := payloadObj["Payload"].(map[string]interface{}); ok {
		if rec, ok := payload["record"]; ok {
			medicalRecord = rec
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"txId":   txID,
		"payload": payloadObj,
		"medical_record": medicalRecord,
	})
}
