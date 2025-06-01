package server

import (
	"encoding/json"
	"unicareos/core/chain"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"io"

	block "unicareos/core/block"
	"unicareos/core/networking"
	"unicareos/core/storage"
	"unicareos/types/ids"
	"unicareos/core/mempool"
	"github.com/golang-jwt/jwt/v5"

	log "log"

	// DEV: Import dev_tx_inspect for dev-only API


	// Load env vars from Dummy.env for local/dev
	_ "github.com/joho/godotenv/autoload"
)

import "github.com/joho/godotenv"

func init() {
	godotenv.Load("Dummy.env")
}

// --- Environment Variable Config ---
// All sensitive/configurable values are loaded from environment variables.
// See Dummy.env for variable names and dummy values.

var (
	apiKey         = os.Getenv("API_KEY")           // API key for admin/peer endpoints
	jwtSecret      = os.Getenv("JWT_SECRET")        // JWT secret for authentication
	dbURL          = os.Getenv("DB_URL")            // Database URL
	awsKmsKeyID    = os.Getenv("AWS_KMS_KEY_ID")    // AWS KMS Key ID
	awsRegion      = os.Getenv("AWS_REGION")        // AWS Region
	s3BucketName   = os.Getenv("S3_BUCKET_NAME")    // S3 Bucket name
	corsOrigins    = os.Getenv("CORS_ALLOWED_ORIGINS") // CORS allowed origins (default: localhost)
	serverPort     = os.Getenv("SERVER_PORT")        // Server port (default: 8080)
	rateLimitPerMin= os.Getenv("RATE_LIMIT_PER_MIN") // Requests per minute per IP/user
	enableHTTPS     = os.Getenv("ENABLE_HTTPS")       // Enable HTTPS (true/false)
	tlsCertPath    = os.Getenv("TLS_CERT_PATH")      // TLS certificate path
	tlsKeyPath     = os.Getenv("TLS_KEY_PATH")       // TLS key path
	logLevel       = os.Getenv("LOG_LEVEL")          // Logging level
	envMode        = os.Getenv("ENV")                // Environment (development/production)
)

// --- Auth Middleware Scaffolding ---
// These functions currently only log warnings; enable enforcement later.

// Checks for API key in X-API-Key header. Logs warning if missing or invalid.
func requireAPIKey(w http.ResponseWriter, r *http.Request) bool {
	key := r.Header.Get("X-API-Key")
	if key == "" {
		log.Println("[WARN] No API key provided (TODO: enforce in prod)")
		return false
	}
	if key != apiKey {
		log.Printf("[WARN] Invalid API key: %s (TODO: enforce in prod)\n", key)
		return false
	}
	return true
}

// Checks for JWT in Authorization: Bearer header. Logs warning if missing or invalid.
func requireJWT(w http.ResponseWriter, r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		log.Println("[WARN] No Bearer token provided (TODO: enforce in prod)")
		return false
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		log.Printf("[WARN] Invalid JWT: %v (TODO: enforce in prod)\n", err)
		return false
	}
	return true
}

// --- Endpoint Auth Markers ---
// TODO: Enforce requireAPIKey on all peer/masternode endpoints (e.g., /request_block, /sync_tip, /broadcast_block, etc.)
// TODO: Enforce requireJWT on all lightweight node endpoints (e.g., /api/cli/submit_memory, /api/cli/mempool, etc.)
// TODO: Remove or restrict all admin/dev endpoints before production


type Server struct {
	store        *storage.Storage
	network      *networking.Network
	ListenAddr   string
	gossipEngine *mempool.GossipEngine
	forkChoice   *chain.ForkChoice
	Finalizer    *block.Finalizer // Added for medical record finalization
}

// --- Ban Event Pool (in-memory, for pending inclusion in next block) ---
var pendingBanEvents []block.BanEvent
var pendingBanEventsLock sync.Mutex


func NewServer(store *storage.Storage, network *networking.Network, listenAddr string, gossipEngine *mempool.GossipEngine, forkChoice *chain.ForkChoice, finalizer *block.Finalizer) *Server {
	return &Server{
		store:        store,
		network:      network,
		ListenAddr:   listenAddr,
		gossipEngine: gossipEngine,
		forkChoice:   forkChoice,
		Finalizer:    finalizer,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/gossip_tx", s.handleGossipTx) // Gossip endpoint
	// Inside Start()
	http.HandleFunc("/connect_peer", s.handleConnectPeer)
	// Modular health/status endpoints
	http.HandleFunc("/nodehealth", s.HandleNodeHealth) // For CLI metrics
	http.HandleFunc("/health/liveness", s.HandleLiveness)
	http.HandleFunc("/health/readiness", s.HandleReadiness)
	http.HandleFunc("/status", s.HandleStatus)
	// Legacy endpoints
	http.HandleFunc("/chain_height", s.handleChainHeight)
	http.HandleFunc("/get_block/", s.handleGetBlock) 
	http.HandleFunc("/list_blocks", s.handleListBlocks)
	http.HandleFunc("/submit_memory", s.handleSubmitMemory)
	http.HandleFunc("/get_status", s.handleStatus)
	http.HandleFunc("/get_chain_tip", s.handleGetChainTip) 
	http.HandleFunc("/check_peers", s.handleCheckPeers)
	// ...
	http.HandleFunc("/request_block", networking.RequestBlockHandler(s.store))
	http.HandleFunc("/sync_tip", s.handleSyncTip)
	http.HandleFunc("/blocks", s.handleBlocksQuery) // New flexible batch/filtered endpoint

	// === Ban Event Admin Endpoint ===
	http.HandleFunc("/admin/ban_event", s.handleAdminBanEvent)

	// === Live sync: block broadcast endpoint ===
	http.HandleFunc("/broadcast_block", s.network.HandleBroadcastBlock)

	// === CLI-specific JSON endpoints ===
	http.HandleFunc("/api/cli/status", s.handleCLIStatus)
	http.HandleFunc("/api/cli/mempool", s.handleCLIMempool)
	http.HandleFunc("/api/cli/submit_memory", s.handleCLISubmitMemory)

	// === Medical Record Submission Endpoint ===
	RegisterMedicalRecordAPI(http.DefaultServeMux, s)

	// === DEV ONLY: Transaction Inspection Endpoint ===
	//Dev delete upon production migration
	RegisterDevTxInspectAPI(http.DefaultServeMux, s)

	fmt.Println("API server listening at", s.ListenAddr)

	enableHTTPS := os.Getenv("ENABLE_HTTPS")
	certPath := os.Getenv("TLS_CERT_PATH")
	keyPath := os.Getenv("TLS_KEY_PATH")

	if enableHTTPS == "true" {
		fmt.Println("[HTTPS] Enabled. Using cert:", certPath, "key:", keyPath)
		return http.ListenAndServeTLS(s.ListenAddr, certPath, keyPath, nil)
	} else {
		fmt.Println("[HTTPS] Disabled. Serving HTTP only!")
		return http.ListenAndServe(s.ListenAddr, nil)
	}
}

// handleGossipTx handles incoming gossip messages
func (s *Server) handleGossipTx(w http.ResponseWriter, r *http.Request) {
    fmt.Println("[GOSSIP] /gossip_tx endpoint hit")
    // Log the remote address and method
    fmt.Printf("[GOSSIP] Request from %s, method: %s\n", r.RemoteAddr, r.Method)
    data, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "invalid body", http.StatusBadRequest)
        fmt.Println("[GOSSIP] Error reading body:", err)
        return
    }
    fmt.Printf("[GOSSIP] Raw body: %s\n", string(data))
    if r.Method != http.MethodPost {
        http.Error(w, "invalid method", http.StatusMethodNotAllowed)
        return
    }
    if s.gossipEngine != nil {
        s.gossipEngine.ReceiveGossip(data)
    }
    w.WriteHeader(http.StatusOK)
}

// handleAdminBanEvent allows an admin to submit a BanEvent (ban or unban) via POST.
func (s *Server) handleAdminBanEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}
	var banEvent block.BanEvent
	err := json.NewDecoder(r.Body).Decode(&banEvent)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	// Optionally, add admin authentication here (e.g., API key, session, etc.)
	banEvent.Timestamp = time.Now().UTC()
	pendingBanEventsLock.Lock()
	pendingBanEvents = append(pendingBanEvents, banEvent)
	pendingBanEventsLock.Unlock()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("BanEvent queued for inclusion in next block"))
}

func (s *Server) handleSubmitMemory(w http.ResponseWriter, r *http.Request) {
	// --- Ban & Rate Limit Enforcement (by IP only) ---
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr // fallback if parsing fails
	}
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		fmt.Printf("[BAN] Blocked submit_memory from banned peer: %s\n", host)
		return
	}
	allowed := s.network.AllowPeerRequest(host)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		fmt.Printf("[RATE LIMIT] Blocked submit_memory from %s\n", host)
		return
	}
	// --- Parse and validate payload ---
	var memSub block.MemorySubmission
	if err := json.NewDecoder(r.Body).Decode(&memSub); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := block.ValidateMemoryPayload(memSub); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// --- Construct and submit transaction ---
	payloadBytes, err := json.Marshal(memSub)
	if err != nil {
		http.Error(w, "failed to marshal memory submission", http.StatusInternalServerError)
		return
	}
	tx := mempool.Transaction{
		TxID:      "memory-" + memSub.Author + "-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
		Sender:    memSub.Author,
	}
	if s.gossipEngine != nil {
		s.gossipEngine.BroadcastTx(tx)
	}
	if s.network.Mempool != nil {
		added := s.network.Mempool.AddTx(tx)
		if !added {
			http.Error(w, "failed to add transaction to mempool", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Memory submission accepted into mempool for inclusion in next block."))
}


// --- CLI-specific JSON endpoints ---
// Returns node status as JSON
func (s *Server) handleCLIStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	height, err := s.forkChoice.Store.GetChainHeight()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "failed to get chain height",
		})
		return
	}
	status := map[string]interface{}{
		"name": "UniCareOS Node",
		"status": "healthy",
		"height": height,
	}
	json.NewEncoder(w).Encode(status)
}

// Returns mempool as JSON array
func (s *Server) handleCLIMempool(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var mempoolTxs []interface{}
	if s.gossipEngine != nil && s.gossipEngine.Mempool != nil {
		txs := s.gossipEngine.Mempool.GetAllTxs()
		for _, tx := range txs {
			mempoolTxs = append(mempoolTxs, tx)
		}
	} else {
		mempoolTxs = []interface{}{}
	}
	json.NewEncoder(w).Encode(mempoolTxs)
}

// Accepts POST, always returns JSON
func (s *Server) handleCLISubmitMemory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}
	var memSub block.MemorySubmission
	if err := json.NewDecoder(r.Body).Decode(&memSub); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}
	if err := block.ValidateMemoryPayload(memSub); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if s.gossipEngine == nil || s.gossipEngine.Mempool == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "mempool unavailable"})
		return
	}
	// Construct tx with the validated memory submission
	payloadBytes, err := json.Marshal(memSub)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to marshal memory submission"})
		return
	}
	tx := mempool.Transaction{
		TxID:      "memory-" + memSub.Author + "-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
		Sender:    memSub.Author,
	}
	added := s.gossipEngine.Mempool.AddTx(tx)
	if !added {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "duplicate or mempool full"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"result": "memory submitted"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { host = r.RemoteAddr }
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		return
	}
	allowed := s.network.AllowPeerRequest(host)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("UniCareOS node is alive"))
}

// ✅ FULL get_chain_tip API
func (s *Server) handleGetChainTip(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { host = r.RemoteAddr }
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		return
	}
	allowed := s.network.AllowPeerRequest(host)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	tip := s.network.GetLatestBlockID()
	tipStr := hex.EncodeToString(tip[:]) // Convert [32]byte → []byte → hex string

	response := map[string]string{
		"latestBlockID": tipStr,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}



// ✅ New: handleGetBlock


func (s *Server) handleGetBlock(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { host = r.RemoteAddr }
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		return
	}
	allowed := s.network.AllowPeerRequest(host)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	blockIDHex := strings.TrimPrefix(r.URL.Path, "/get_block/")
	if blockIDHex == "" {
		http.Error(w, "block ID required", http.StatusBadRequest)
		return
	}

	blockIDBytes, err := hex.DecodeString(blockIDHex)
	if err != nil {
		http.Error(w, "invalid block ID format", http.StatusBadRequest)
		return
	}

	var blockID ids.ID
	if len(blockIDBytes) != len(blockID) {
		http.Error(w, "invalid block ID length", http.StatusBadRequest)
		return
	}
	copy(blockID[:], blockIDBytes)

	blk, err := s.store.GetBlock(blockID[:]) // ✅ NOTE: blockID[:] here
	if err != nil {
		http.Error(w, fmt.Sprintf("block not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blk)
}

// ✅ New: handleListBlocks
func (s *Server) handleListBlocks(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { host = r.RemoteAddr }
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		return
	}
	allowed := s.network.AllowPeerRequest(host)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	// Fetch the last N blocks from storage
	const maxBlocks = 10
	blockSummaries, err := s.store.ListRecentBlocks(maxBlocks)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not list blocks: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blockSummaries)
}

// ✅ New: handleChainHeight
func (s *Server) handleChainHeight(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { host = r.RemoteAddr }
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		return
	}
	allowed := s.network.AllowPeerRequest(host)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	height, err := s.store.GetChainHeight()
	if err != nil {
		http.Error(w, fmt.Sprintf("could not determine chain height: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]int{
		"chainHeight": height,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ✅ New: handleHealth
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { host = r.RemoteAddr }
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		return
	}
	allowed := s.network.AllowPeerRequest(host)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	height, err := s.store.GetChainHeight()
	if err != nil {
		http.Error(w, fmt.Sprintf("could not determine chain height: %v", err), http.StatusInternalServerError)
		return
	}

	tipID, err := s.store.GetLatestBlockID()
	if err != nil {
		http.Error(w, fmt.Sprintf("could not load latest block ID: %v", err), http.StatusInternalServerError)
		return
	}

	tipBlockBytes, err := s.store.GetBlock(tipID[:])
	if err != nil {
		http.Error(w, fmt.Sprintf("could not load tip block: %v", err), http.StatusInternalServerError)
		return
	}

	var tipBlock block.Block
	err = json.Unmarshal(tipBlockBytes, &tipBlock)
	if err != nil {
		http.Error(w, "invalid tip block structure", http.StatusInternalServerError)
		return
	}

	tipAge := time.Since(tipBlock.Timestamp).Seconds()

	response := map[string]interface{}{
		"status":           "healthy",
		"chainHeight":      height,
		"tipBlockID":       fmt.Sprintf("%x", tipID[:]),
		"lastBlockTimeUTC": tipBlock.Timestamp.UTC().Format(time.RFC3339),
		"tipAgeSeconds":    int(tipAge),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ✅ New: handleConnectPeer
func (s *Server) handleConnectPeer(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { host = r.RemoteAddr }
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		return
	}
	allowed := s.network.AllowPeerRequest(host)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Address string `json:"address"`
		APIPort int    `json:"apiPort"`
	}

	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil || payload.Address == "" {
		http.Error(w, "invalid payload or missing address", http.StatusBadRequest)
		return
	}

	err = s.network.ConnectToPeer(payload.Address)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect: %v", err), http.StatusInternalServerError)
		return
	}
	// Automatically trigger fork-choice sync from the new peer
	go func() {
		fmt.Printf("[AUTO_SYNC] Triggering fork-choice sync from peer %s\n", payload.Address)
		// Fetch peer chain height
		peerHeight := 0
		peerTip := [32]byte{}
		resp, err := http.Get("http://" + payload.Address + "/chain_height")
		if err == nil && resp.StatusCode == 200 {
			var result struct{ ChainHeight int `json:"chainHeight"` }
			json.NewDecoder(resp.Body).Decode(&result)
			peerHeight = result.ChainHeight
		}
		if resp != nil { resp.Body.Close() }
		resp, err = http.Get("http://" + payload.Address + "/get_chain_tip")
		if err == nil && resp.StatusCode == 200 {
			var result struct{ LatestBlockID string `json:"latestBlockID"` }
			json.NewDecoder(resp.Body).Decode(&result)
			if len(result.LatestBlockID) == 64 {
				blockIDBytes, _ := hex.DecodeString(result.LatestBlockID)
				copy(peerTip[:], blockIDBytes)
			}
		}
		if resp != nil { resp.Body.Close() }
		myHeight, _ := s.store.GetChainHeight()
		myTip := s.network.GetLatestBlockID()
		// Always use the peer's API port for fork-choice sync
		apiHost := payload.Address
		if colonIdx := strings.LastIndex(apiHost, ":"); colonIdx != -1 {
			apiHost = apiHost[:colonIdx]
		}
		apiAddr := fmt.Sprintf("%s:%d", apiHost, payload.APIPort)
		peers := []chain.PeerTipInfo{{ Height: peerHeight, BlockID: peerTip, Address: apiAddr }}
		err = s.forkChoice.CheckAndSync(myHeight, myTip, peers)
		if err != nil {
			fmt.Printf("[AUTO_SYNC] Fork-choice sync error from peer %s: %v\n", payload.Address, err)
		} else {
			fmt.Printf("[AUTO_SYNC] Fork-choice sync completed from peer %s\n", payload.Address)
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Successfully connected to peer: %s (sync started)", payload.Address)))
}

// ✅ New: handleCheckPeers
func (s *Server) handleCheckPeers(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { host = r.RemoteAddr }
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		return
	}
	allowed := s.network.AllowPeerRequest(host)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	report := s.network.CheckPeerTips()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// ✅ NEW: handleRequestBlock
func (s *Server) handleRequestBlock(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { host = r.RemoteAddr }
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		return
	}
	allowed := s.network.AllowPeerRequest(host)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		BlockID string `json:"blockID"`
	}
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil || payload.BlockID == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	blockIDBytes, err := hex.DecodeString(payload.BlockID)
	if err != nil || len(blockIDBytes) != 32 {
		http.Error(w, "invalid block ID format", http.StatusBadRequest)
		return
	}

	fmt.Printf("[REQUEST_BLOCK] Requested blockID: %x\n", blockIDBytes)

	blockData, err := s.store.GetBlock(blockIDBytes)
	if err != nil {
		fmt.Printf("[REQUEST_BLOCK] Block not found for ID %x: %v\n", blockIDBytes, err)
		http.Error(w, "block not found", http.StatusNotFound)
		return
	}

	fmt.Printf("[REQUEST_BLOCK] Block as string: %s\n", string(blockData))

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(blockData)
}

func (s *Server) handleSyncTip(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { host = r.RemoteAddr }
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		return
	}
	allowed := s.network.AllowPeerRequest(host)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	var payload struct {
		PeerAddress string `json:"peer"`
		BlockID     string `json:"blockID"`
	}

	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil || payload.PeerAddress == "" || payload.BlockID == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	blockIDBytes, err := hex.DecodeString(payload.BlockID)
	if err != nil || len(blockIDBytes) != 32 {
		http.Error(w, "invalid block ID", http.StatusBadRequest)
		return
	}

	var blockID [32]byte
	copy(blockID[:], blockIDBytes)

	err = s.network.SyncFromPeer(payload.PeerAddress, blockID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Synced block from peer"))
}

// handleBlocksQuery provides batch, filtered, and paginated block retrieval via /blocks endpoint
func (s *Server) handleBlocksQuery(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil { host = r.RemoteAddr }
	if s.network.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		return
	}
	allowed := s.network.AllowPeerRequest(host)

	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
    // Parse query params: start, end (heights), validator (optional), limit, offset
    start := 0
    end := -1 // means 'to tip'
    validator := ""
    limit := 10  // default limit
    offset := 0  // default offset
    maxLimit := 100

    q := r.URL.Query()
    if v := q.Get("start"); v != "" {
        start, err = strconv.Atoi(v)
        if err != nil || start < 0 {
            http.Error(w, "invalid start height", http.StatusBadRequest)
            return
        }
    }
    if v := q.Get("end"); v != "" {
        end, err = strconv.Atoi(v)
        if err != nil || end < start {
            http.Error(w, "invalid end height", http.StatusBadRequest)
            return
        }
    }
    if v := q.Get("validator"); v != "" {
        validator = v
    }
    if v := q.Get("limit"); v != "" {
        limit, err = strconv.Atoi(v)
        if err != nil || limit <= 0 {
            http.Error(w, "invalid limit", http.StatusBadRequest)
            return
        }
        if limit > maxLimit {
            limit = maxLimit
        }
    }
    if v := q.Get("offset"); v != "" {
        offset, err = strconv.Atoi(v)
        if err != nil || offset < 0 {
            http.Error(w, "invalid offset", http.StatusBadRequest)
            return
        }
    }

    chainHeight, err := s.store.GetChainHeight()
    if err != nil {
        http.Error(w, "failed to get chain height", http.StatusInternalServerError)
        return
    }
    // Determine range
    if end == -1 || end >= chainHeight {
        end = chainHeight - 1
    }
    if start > end {
        http.Error(w, "start height exceeds end height", http.StatusBadRequest)
        return
    }
    // Apply offset and limit
    paginatedStart := start + offset
    if paginatedStart > end {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode([]block.Block{})
        return
    }
    paginatedEnd := paginatedStart + limit - 1
    if paginatedEnd > end {
        paginatedEnd = end
    }

    var blocks []block.Block
    for h := paginatedStart; h <= paginatedEnd; h++ {
        blk, err := s.store.GetBlockByHeight(h)
        if err != nil {
            continue // skip missing blocks
        }
        if validator != "" && blk.ValidatorDID != validator {
            continue
        }
        blocks = append(blocks, blk)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(blocks)
}  