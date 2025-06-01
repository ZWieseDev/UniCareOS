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
	"unicareos/core/block"
	"unicareos/core/validation"
	"unicareos/core/auth"
	"unicareos/core/mempool"
)
var Authorizer *auth.Authorizer
var EthosVerifier *auth.EthosVerifier // Decoupled Ethos verification (exported)
// getAPISecret fetches the API secret/token from env
func getAPISecret() string {
	return os.Getenv("API_JWT_SECRET") // Set this in Dummy.env
}

// Middleware for JWT/API key authentication
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := getAPISecret()
		authHeader := r.Header.Get("Authorization")
		// DEBUG PRINTS
		//println("[DEBUG] Loaded API_JWT_SECRET:", secret)
		//println("[DEBUG] Incoming Authorization header:", authHeader)
		if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != secret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Handler for submitting medical records
func (s *Server) SubmitMedicalRecordHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[DEBUG] /submit-medical-record handler called")
	bodyBytes, _ := io.ReadAll(r.Body)
	//fmt.Printf("[DEBUG] Incoming request body: %s\n", string(bodyBytes))
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset body for decoding
	var submission block.MedicalRecordSubmission
	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		fmt.Printf("[DEBUG] JSON decode error: %v\n", err)
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if submission.SubmissionTimestamp.IsZero() {
		submission.SubmissionTimestamp = time.Now().UTC()
	}
	fmt.Println("[DEBUG] Calling VerifyWalletSignature...")
	verr := validation.VerifyWalletSignature(submission.Record, submission.Signature, submission.WalletAddress)
	if verr != nil {
		fmt.Printf("[DEBUG] Signature/allowlist verification failed: %v\n", verr)
		http.Error(w, "Unauthorized: "+verr.Error(), http.StatusUnauthorized)
		return
	}
	fmt.Println("[DEBUG] Signature/allowlist verification succeeded")

	// --- DECOUPLED: Ethos Token Verification (independent of wallet logic) ---
	ethosToken := r.Header.Get("X-Ethos-Token")
	if ethosToken == "" {
		fmt.Println("[DEBUG] Missing Ethos token in X-Ethos-Token header")
		http.Error(w, "Missing Ethos token (X-Ethos-Token header required)", http.StatusUnauthorized)
		return
	}
	// Use a global or package-level EthosVerifier (not Authorizer) for Ethos-only verification
	if EthosVerifier != nil {

		claims, err := EthosVerifier.VerifyEthosToken(ethosToken)
		if err != nil {
			fmt.Printf("[DEBUG] Ethos token verification failed: %v\n", err)
			http.Error(w, "Unauthorized (Ethos token): "+err.Error(), http.StatusUnauthorized)
			return
		}
		fmt.Printf("[DEBUG] Ethos token verification succeeded: %+v\n", claims)
	} else {
		fmt.Println("[DEBUG] Ethos verifier not initialized, skipping Ethos token verification (DEV ONLY)")
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

	// --- Finalization logic ---
	finalizerPubKeyB64 := os.Getenv("FINALIZER_PUBKEY")
	if finalizerPubKeyB64 != "" && s.Finalizer != nil {
		fmt.Println("\033[34m[FINALIZER] Attempting to finalize event...\033[0m")
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
			fmt.Printf("\033[34m[FINALIZER ERROR] %v\033[0m\n", err)
		} else {
			fmt.Println("\033[34m[FINALIZER] Event finalized and logged.\033[0m")
		}
	} else {
		fmt.Println("[FINALIZER] Finalizer or pubkey not set, skipping finalization.")
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

func RegisterMedicalRecordAPI(mux *http.ServeMux, server *Server) {
	mux.Handle("/api/v1/submit-medical-record", authMiddleware(http.HandlerFunc(server.SubmitMedicalRecordHandler)))
	mux.HandleFunc("/api/v1/expired-medical-records", server.ListExpiredMedicalRecordsHandler)
	mux.HandleFunc("/api/v1/resubmit-medical-record", server.ResubmitMedicalRecordHandler)
}
