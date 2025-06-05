package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"bytes"
	"fmt"
	"crypto/sha256"
	"net"
	"unicareos/types/ids"
	"unicareos/core/block"
	"unicareos/core/validation"
	"unicareos/core/auth"
	"unicareos/core/mempool"
	"unicareos/core/audit"
)

// Helper to convert []interface{} to []block.MemorySubmission
func convertMemories(raw []interface{}) []block.MemorySubmission {
	var out []block.MemorySubmission
	for _, mem := range raw {
		if m, ok := mem.(block.MemorySubmission); ok {
			out = append(out, m)
			continue
		}
		if m, ok := mem.(map[string]interface{}); ok {
			b, _ := json.Marshal(m)
			var ms block.MemorySubmission
			if err := json.Unmarshal(b, &ms); err == nil {
				out = append(out, ms)
			}
		}
	}
	return out
}

var Authorizer *auth.Authorizer
var EthosVerifier *auth.EthosVerifier // Decoupled Ethos verification (exported)
// getAPISecret fetches the API secret/token from env
func getAPISecret() string {
	return os.Getenv("API_JWT_SECRET") // Set this in Dummy.env
}

// Middleware for JWT/API key authentication (enforce either JWT or API key)
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwtSecret := getAPISecret()
		apiKey := os.Getenv("API_KEY")
		authHeader := r.Header.Get("Authorization")
		xApiKey := r.Header.Get("X-API-Key")

		jwtValid := false
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == jwtSecret && token != "" {
				jwtValid = true
			}
		}
		apiKeyValid := (xApiKey != "" && apiKey != "" && xApiKey == apiKey)

		if !jwtValid && !apiKeyValid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Handler for submitting medical records
func (s *Server) SubmitMedicalRecordHandler(w http.ResponseWriter, r *http.Request) {

	bodyBytes, _ := io.ReadAll(r.Body)

	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset body for decoding
	var submission block.MedicalRecordSubmission
	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {

		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if submission.SubmissionTimestamp.IsZero() {
		submission.SubmissionTimestamp = time.Now().UTC()
	}

	verr := validation.VerifyWalletSignature(submission.Record, submission.Signature, submission.WalletAddress)
	if verr != nil {

		http.Error(w, "Unauthorized: "+verr.Error(), http.StatusUnauthorized)
		return
	}


	// --- DECOUPLED: Ethos Token Verification (independent of wallet logic) ---
	ethosToken := r.Header.Get("X-Ethos-Token")
	if ethosToken == "" {

		http.Error(w, "Missing Ethos token (X-Ethos-Token header required)", http.StatusUnauthorized)
		return
	}
	// Use a global or package-level EthosVerifier (not Authorizer) for Ethos-only verification
	if EthosVerifier != nil {

		_, err := EthosVerifier.VerifyEthosToken(ethosToken)
		if err != nil {

			http.Error(w, "Unauthorized (Ethos token): "+err.Error(), http.StatusUnauthorized)
			return
		}

	} else {

	}

	// Serialize the validated submission
	serializedPayload, err := json.Marshal(submission)
	if err != nil {
		http.Error(w, "Failed to serialize submission", http.StatusInternalServerError)
		return
	}

	// Generate TxID (SHA256 of payload)
	hash := sha256.Sum256(serializedPayload)
	txID := fmt.Sprintf("%x", hash[:])

	tx := mempool.Transaction{
		TxID:      txID,
		Payload:   serializedPayload,
		Timestamp: time.Now().Unix(),
		Sender:    submission.WalletAddress,
	}

	added := s.gossipEngine.Mempool.AddTx(tx)
	if !added {
		http.Error(w, "Duplicate transaction or mempool full", http.StatusConflict)
		return
	}

	// --- Revision Audit Logging (additive, non-invasive) ---
	if submission.RevisionOf != "" || submission.RevisionReason != "" || (submission.DocLineage != nil && len(submission.DocLineage) > 0) {
		// Import audit package at top if not already: "unicareos/core/audit"
		auditLogger := audit.NewStdoutAuditLogger() // Replace with your logger if needed
		eventID := txID
		revisionOf := submission.RevisionOf
		revisionReason := submission.RevisionReason
		docLineage := submission.DocLineage
		entityID := submission.WalletAddress
		result := "success"
		audit.LogMedicalRecordRevision(auditLogger, eventID, revisionOf, revisionReason, entityID, docLineage, result)
	}

	// --- Finalization logic ---
	finalizerPubKeyB64 := os.Getenv("FINALIZER_PUBKEY")
	if finalizerPubKeyB64 != "" && s.Finalizer != nil {
		finalizeTx := &block.FinalizeEventTx{
			TxID:                  txID,
			SubmitMedicalRecordTx: tx.Payload,
			FinalizerSignature:    "", // Will be set by Finalizer logic
			EthosToken:            ethosToken,
			Block: block.BlockReference{
				BlockHash: txID, // Use txID as a non-empty stand-in for testing
				Epoch:     0,  // Set as appropriate
			},
			Timestamp: time.Now().UTC(),
			Status:    block.FinalizationStatusPending,
		}
		err := s.Finalizer.FinalizeEvent(finalizeTx, finalizerPubKeyB64)
		if err != nil {
			// Optionally, handle/log the error elsewhere if needed
		}
		// No debug/info output
	}

	// Return a receipt
	receipt := map[string]interface{}{
		"txId":    txID,
		"status":  "pending",
		"message": "Submission added to mempool",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(receipt)
}

// RegisterMedicalRecordAPI registers the endpoint to the mux
// Handler to list all expired medical record transactions
func (s *Server) ListExpiredMedicalRecordsHandler(w http.ResponseWriter, r *http.Request) {
	// For demo: no auth, but can add
	expired := s.gossipEngine.Mempool.ExpiredPool.ListExpiredTxs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(expired)
}

// Handler to resubmit an expired medical record transaction
func (s *Server) ResubmitMedicalRecordHandler(w http.ResponseWriter, r *http.Request) {
	var req struct { TxID string `json:"txId"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TxID == "" {
		http.Error(w, "Missing or invalid txId", http.StatusBadRequest)
		return
	}
	empTx, ok := s.gossipEngine.Mempool.ExpiredPool.GetExpiredTx(req.TxID)
	if !ok {
		http.Error(w, "Transaction not found in expired pool", http.StatusNotFound)
		return
	}
	// Type assertion for []byte
	payloadBytes, ok := empTx.Payload.([]byte)
	if !ok {
		http.Error(w, "Expired TX payload is not []byte", http.StatusInternalServerError)
		return
	}
	// Re-insert into mempool with new timestamp
	var submission map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &submission); err != nil {
		http.Error(w, "Failed to parse expired TX payload", http.StatusInternalServerError)
		return
	}

	tx := mempool.Transaction{
		TxID:      req.TxID + "-resubmitted-" + time.Now().Format("20060102150405"),
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
		Sender:    "resubmitted",
	}
	added := s.gossipEngine.Mempool.AddTx(tx)
	if !added {
		http.Error(w, "Duplicate or mempool full", http.StatusConflict)
		return
	}
	// Audit log print
	fmt.Printf("[AUDIT] Resubmitted expired TX: %s at %s\n", req.TxID, time.Now().Format(time.RFC3339))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"txId": tx.TxID,
		"status": "resubmitted",
		"message": "Transaction resubmitted to mempool",
	})
}

func (s *Server) GetLineageHandler(w http.ResponseWriter, r *http.Request) {
	// --- AUDIT LOGGING ---
	getRequester := func(r *http.Request) string {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if len(token) > 0 {
				return "jwt:" + token[:8] // partial for privacy
			}
		}
		apikey := r.Header.Get("X-API-Key")
		if apikey != "" {
			return "apikey:" + apikey[:8]
		}
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil { return host }
		return r.RemoteAddr
	}
	auditLog := func(queriedBy, recordId, status, reason string) {
		entry := map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"queriedBy": queriedBy,
			"recordId":  recordId,
			"status":    status,
			"reason":    reason,
		}
		f, err := os.OpenFile("audit.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err == nil {
			json.NewEncoder(f).Encode(entry)
			f.Close()
		}
	}
	queriedBy := getRequester(r)


	eventType := r.URL.Query().Get("eventType")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	authorValidator := r.URL.Query().Get("authorValidator")

	var fromTime, toTime time.Time
	var fromSet, toSet bool
	var err error
	if fromStr != "" {
		fromTime, err = time.Parse("2006-01-02", fromStr)
		if err == nil { fromSet = true }
	}
	if toStr != "" {
		toTime, err = time.Parse("2006-01-02", toStr)
		if err == nil { toSet = true }
	}

	eventId := r.URL.Query().Get("eventId")
	if eventId == "" {
		auditLog(queriedBy, "", "failure", "Missing eventId parameter")
		http.Error(w, "Missing eventId parameter", http.StatusBadRequest)
		return
	}
	if s.store == nil {
		auditLog(queriedBy, eventId, "failure", "No storage backend")
		http.Error(w, "No storage backend", http.StatusInternalServerError)
		return
	}
	height, err := s.store.GetChainHeight()
	if err != nil {
		auditLog(queriedBy, eventId, "failure", "Failed to get chain height")
		http.Error(w, "Failed to get chain height", http.StatusInternalServerError)
		return
	}
	var foundEvent *block.ChainedEvent
	for i := 0; i < height; i++ {
		blk, err := s.store.GetBlockByHeight(i)
		if err != nil { continue }
		for i := range blk.Events {
			evt := blk.Events[i]
			tmp := &block.ChainedEvent{
				RecordID: evt.RecordID,
				EventID: evt.EventID,
				EventType: evt.EventType,
				Description: evt.Description,
				Timestamp: evt.Timestamp,
				AuthorValidator: evt.AuthorValidator,
				Memories: convertMemories(evt.Memories),
				PatientID: evt.PatientID,
				ProviderID: evt.ProviderID,
				Epoch: evt.Epoch,
				PayloadHash: evt.PayloadHash,
				PayloadRef: evt.PayloadRef,
				RevisionReason: evt.RevisionReason,
				RevisionOf: evt.RevisionOf,
				DocLineage: evt.DocLineage,
			}
			if tmp.EventID.String() == eventId {
				foundEvent = tmp
				break
			}
		}
		if foundEvent != nil { break }
	}
	if foundEvent == nil {
		auditLog(queriedBy, eventId, "failure", "eventId not found")
		http.Error(w, "eventId not found", http.StatusNotFound)
		return
	}
	// Build full lineage recursively, applying filters
	lineage := []string{}
	visited := make(map[string]bool)
	currentID := foundEvent.RevisionOf
	for len(currentID) > 0 && !visited[currentID] {
		visited[currentID] = true
		var ancestor *block.ChainedEvent
		for i := 0; i < height; i++ {
			blk, err := s.store.GetBlockByHeight(i)
			if err != nil { continue }
			for i := range blk.Events {
				evt := blk.Events[i]
				tmp := &block.ChainedEvent{
					EventID: evt.EventID,
					RecordID: evt.RecordID,
					EventType: evt.EventType,
					Description: evt.Description,
					Timestamp: evt.Timestamp,
					AuthorValidator: evt.AuthorValidator,
					RevisionReason: evt.RevisionReason,
					RevisionOf: evt.RevisionOf,
					DocLineage: evt.DocLineage,
				}
				// --- FILTERS ---
				if eventType != "" && tmp.EventType != eventType {
					continue
				}
				if (fromSet || toSet) && !tmp.Timestamp.IsZero() {
					if fromSet && tmp.Timestamp.Before(fromTime) {
						continue
					}
					if toSet && tmp.Timestamp.After(toTime.Add(24*time.Hour)) {
						continue
					}
				}
				if authorValidator != "" && tmp.AuthorValidator != (ids.ID{}) {
					if strings.ToLower(tmp.AuthorValidator.String()) != strings.ToLower(authorValidator) {
						continue
					}
				}
				// --- END FILTERS ---
				if tmp.EventID.String() == currentID {
					ancestor = tmp
					break
				}
			}
			if ancestor != nil { break }
		}
		if ancestor == nil {
			break
		}
		if ancestor.DocLineage != nil {
			lineage = append(lineage, ancestor.DocLineage...)
		}
		lineage = append(lineage, currentID)
		currentID = ancestor.RevisionOf
	}
	// Reverse to chronological order
	for i, j := 0, len(lineage)-1; i < j; i, j = i+1, j-1 {
		lineage[i], lineage[j] = lineage[j], lineage[i]
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"eventId": eventId,
		"docLineage": lineage,
		"recordId": foundEvent.RecordID,
		"revisionOf": foundEvent.RevisionOf,
		"revisionReason": foundEvent.RevisionReason,
		"event": foundEvent,
	})
	auditLog(queriedBy, eventId, "success", "ok")
}

func RegisterMedicalRecordAPI(mux *http.ServeMux, server *Server) {
	mux.Handle("/api/v1/submit-medical-record", authMiddleware(http.HandlerFunc(server.SubmitMedicalRecordHandler)))
	mux.HandleFunc("/api/v1/expired-medical-records", server.ListExpiredMedicalRecordsHandler)
	mux.HandleFunc("/api/v1/resubmit-medical-record", server.ResubmitMedicalRecordHandler)
	mux.Handle("/api/v1/get-lineage", authMiddleware(http.HandlerFunc(server.GetLineageHandler)))
}
