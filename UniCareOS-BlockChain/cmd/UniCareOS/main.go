package main

import (
	"fmt"
	"unicareos/core/state"
	"log"
	"bytes"
	"os"
	"io"
	"strconv"
	"time"
	"net/http"
	"encoding/json"	
	"crypto/ed25519"
    "encoding/base64"	
	"unicareos/api/server"
	"unicareos/core/genesis"
	"unicareos/core/networking"
	"unicareos/core/storage"
	"unicareos/core/block" // ✅ Needed for Deserialize
	"unicareos/core" // For Ed25519 keys and signing
	"unicareos/core/mempool"
	"unicareos/core/chain"
	"unicareos/core/auth"
	"unicareos/core/audit"
	"unicareos/core/scan"
	"strings"
)
// Minimal audit logger for Finalizer
// Implements block.AuditLogger

type FinalizerAuditLogger struct{}

func (l *FinalizerAuditLogger) LogFinalization(txID string, status block.FinalizationStatus, reason string) error {
	fmt.Printf("[FINALIZER AUDIT] txID=%s status=%v reason=%s\n", txID, status, reason)
	return nil
}



// Default block production interval (1 second)
var blockProductionInterval = 3 * time.Second
// Grace period after slot before fallback is allowed
var fallbackGracePeriod = 500 * time.Millisecond // configurable if needed

func init() {
	if val := os.Getenv("BLOCK_TIME_MS"); val != "" {
		if ms, err := strconv.Atoi(val); err == nil {
			blockProductionInterval = time.Duration(ms) * time.Millisecond
		}
	}
}

// isRetryableError returns true if the error is considered transient and worth retrying.
func isRetryableError(err string) bool {
	err = strings.ToLower(err)
	if strings.Contains(err, "timeout") || strings.Contains(err, "network") {
		return true
	}
	// Add more retryable error patterns as needed
	return false // All others are considered non-retryable
}

func main() {
	// Log to file as well as stdout
	logFile, err := os.OpenFile("logs/unicareos-node.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	fmt.Println("🚀 Starting UniCareOS Node")
	scripts.ScanChain()

	// === Node Key Management ===
	pubKey, privKey, err := core.GenerateAndSaveKeypair()
	if err != nil {
		log.Fatalf("❌ Failed to load/generate Ed25519 keypair: %v", err)
	}

	// --- Initialize Authorizer for Ethos token verification (wallet verifier untouched) ---
	// Load Ethos RS256 public key
	ethosPubKey, err := auth.LoadRSAPublicKeyFromFile("ethos_public.pem")
	if err != nil {
		log.Fatalf("Failed to load Ethos public key: %v", err)
	}
	keyProvider := &auth.DummyKeyProvider{
		PublicKey: ethosPubKey,
	}
	// Decoupled: Initialize ethosVerifier for Ethos token logic
	server.EthosVerifier = &auth.EthosVerifier{KeyProvider: keyProvider}
	ethosVerifier := &auth.EthosVerifier{KeyProvider: keyProvider}
	auditLogger := audit.NewStdoutAuditLogger()
	if server.Authorizer == nil {
		server.Authorizer = &auth.Authorizer{
			EthosVerifier: ethosVerifier,
			AuditLogger:   auditLogger,
			// WalletVerifier: untouched; set elsewhere as before
		}
	}

	fmt.Printf("[KEY] Node public key: %x\n", pubKey)

// === Wallet Allowlist Logging ===
func() {
	file, err := os.Open("core/block/authorized_wallets.json")
	if err != nil {
		fmt.Printf("[Wallet Allowlist] Failed to open authorized_wallets.json: %v\n", err)
		return
	}
	defer file.Close()
	var wallets map[string]struct {
		Authorized bool   `json:"authorized"`
		PublicKey  string `json:"publicKey"`
	}
	bytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("[Wallet Allowlist] Failed to read authorized_wallets.json: %v\n", err)
		return
	}
	if err := json.Unmarshal(bytes, &wallets); err != nil {
		fmt.Printf("[Wallet Allowlist] Failed to parse authorized_wallets.json: %v\n", err)
		return
	}
	fmt.Println("[Wallet Allowlist] Loaded wallets at startup:")
	for addr, info := range wallets {
		fmt.Printf("  - %s: authorized=%v, publicKey=%s\n", addr, info.Authorized, info.PublicKey)
	}
}()

	// === Config ===
	dbPath := "./unicareos_db"
	apiListenAddr := ":8080"
	networkListenAddr := ":3000"

	// === Storage ===
	store, err := storage.NewStorage(dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// === Robust tip recovery: scan all blocks for highest ===
	maxHeight := 0
	var tipBlockID [32]byte
	iter := store.DB().NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		if bytes.HasPrefix(key, []byte("block:")) {
			blkBytes := iter.Value()
			decBytes, decErr := storage.Decrypt(blkBytes)
			if decErr != nil {
				fmt.Printf("[RECOVERY][ERROR] Failed to decrypt block for key %s: %v\n", key, decErr)
				fmt.Printf("[RECOVERY][ERROR] Encrypted value (first 32 bytes): %x\n", blkBytes[:32])
				continue
			}
			blk, err := block.Deserialize(decBytes)
			if err != nil {
				fmt.Printf("[RECOVERY][ERROR] Failed to deserialize decrypted block for key %s: %v\n", key, err)
				fmt.Printf("[RECOVERY][ERROR] Decrypted value (first 32 bytes): %x\n", decBytes[:32])
				continue
			}
			if int(blk.Height) > maxHeight {
				maxHeight = int(blk.Height)
				tipBlockID = blk.BlockID
			}
		}
	}
	iter.Release()
	if maxHeight > 0 {
		fmt.Printf("[RECOVERY] Highest block found: height %d, BlockID %x\n", maxHeight, tipBlockID)
		// Set tip in network and storage
		// (network will be initialized next)
	} else {
		fmt.Println("[RECOVERY] No blocks found in DB, will create or use genesis.")
	}

	// === Guard: prevent zeroed block tip sync
	latestID, err := store.GetLatestBlockID()
	if err == nil && latestID == [32]byte{} {
		log.Fatalf("❌ Tip block is all-zero — aborting node startup to prevent corrupt sync.")
	}

	// === Load or Create Genesis Block ===
	genesisExists, _ := store.HasGenesisBlock()
	if !genesisExists {
		fmt.Println("🌟 No genesis block found. Creating Genesis...")
		genesisBlock := genesis.CreateGenesisBlock()
		blockBytes, err := genesisBlock.Serialize()
		if err != nil {
			log.Fatalf("❌ Failed to serialize genesis block: %v", err)
		}
		err = store.SaveBlock(genesisBlock.BlockID[:], blockBytes)
		if err != nil {
			log.Fatalf("❌ Failed to save genesis block: %v", err)
		}
		fmt.Println(" Genesis block created and saved!")
	} else {
		fmt.Println(" Genesis block already exists.")
	}

	// === Load genesis config for epoch settings ===
	genesisCfg, err := genesis.LoadGenesisConfig("genesis.json")
	if err != nil {
		log.Fatalf(" Failed to load genesis config: %v", err)
	}
	epochBlockCount := genesisCfg.InitialParams.EpochBlockCount

	// === Network ===
	// --- Initialize ChainState and load epoch state ---
	chainState := &state.ChainState{StateDB: store}
	if err := chainState.LoadEpochState(); err != nil {
		fmt.Printf("[EPOCH] Failed to load epoch state: %v\n", err)
	}
	network := networking.NewNetwork(networkListenAddr, store, 8080, pubKey, privKey, chainState, epochBlockCount)
	// Set block production interval for networking
	network.BlockProductionInterval = blockProductionInterval


	// === Set recovered tip in network ===
	if maxHeight > 0 {
		if err := network.SetLatestBlockID(tipBlockID); err != nil {
			fmt.Printf("[ERROR] Failed to persist latest block ID: %v\n", err)
		}
		fmt.Printf("[RECOVERY] Set network tip to block %x at height %d\n", tipBlockID, maxHeight)

		// Load the tip block from storage and update epoch state
		blockBytes, err := store.GetBlock(tipBlockID[:])
		if err == nil && blockBytes != nil {
			tipBlockObj, err := block.Deserialize(blockBytes)
			if err == nil && tipBlockObj != nil {
				chainState.Epoch = tipBlockObj.Epoch
				chainState.BlocksInEpoch = tipBlockObj.Height % uint64(epochBlockCount)
				fmt.Printf("[EPOCH] ChainState updated from tip block: Epoch=%d, BlocksInEpoch=%d\n", chainState.Epoch, chainState.BlocksInEpoch)
			} else {
				fmt.Printf("[EPOCH] Could not deserialize tip block for epoch state update: %v\n", err)
			}
		} else {
			fmt.Printf("[EPOCH] Could not load tip block for epoch state update: %v\n", err)
		}
	}

	// === Mempool wiring ===
	mp := mempool.NewMempool(1000) // Main mempool instance
	network.Mempool = mp

	// === Background expiry worker for archiving expired TXs ===
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			<-ticker.C
			before := len(mp.GetAllTxs())
			mp.PurgeExpired(15 * time.Minute)
			after := len(mp.GetAllTxs())
			archived := before - after
			if archived > 0 {
				log.Printf("[MEMPOOL] Archived %d expired transaction(s) to ExpiredTxPool", archived)

				// Auto-resubmit all expired transactions
				expiredTxs := mp.ExpiredPool.ListExpiredTxs()
				for _, expiredTx := range expiredTxs {
					// Only auto-resubmit if this is the original transaction (not a descendant)
					if strings.Contains(expiredTx.TxID, "-auto-resubmitted-") {
						continue // skip all resubmitted descendants
					}
					if expiredTx.ResubmitCount >= 3 {
						log.Printf("[MEMPOOL] Skipping auto-resubmission for %s (limit reached)", expiredTx.TxID)
						continue
					}
					// Smarter error handling: only retry if LastError is retryable
					if expiredTx.LastError != "" && !isRetryableError(expiredTx.LastError) {
						log.Printf("[MEMPOOL] Not retrying %s due to non-retryable error: %s", expiredTx.TxID, expiredTx.LastError)
						continue
					}
					// Resubmit the original transaction with the original TxID
					payloadBytes, ok := expiredTx.Payload.([]byte)
					if !ok {
						// If the payload is not []byte, try to marshal it
						payloadBytes, _ = json.Marshal(expiredTx.Payload)
					}
					resubmittedTx := mempool.Transaction{
						TxID:      expiredTx.TxID,
						Payload:   payloadBytes,
						Timestamp: time.Now().Unix(),
					}
					if mp.AddTx(resubmittedTx) {
						// Update the expiredTx in the pool and persist the new resubmission count
						expiredTx.ResubmitCount++
						mp.ExpiredPool.AddExpiredTx(expiredTx)
						log.Printf("[MEMPOOL] Auto-resubmitted expired tx %s (attempt %d)", expiredTx.TxID, expiredTx.ResubmitCount)
					}
				}
			}
		}
	}()

	err = network.Start()
	if err != nil {
		log.Fatalf("❌ Failed to start networking: %v", err)
	}

	// === 🛠 IMPORTANT: Load latest real block ID into network
	genesisBlockBytes, err := store.GetGenesisBlock()
	if err != nil {
		log.Fatalf("❌ Failed to load genesis block: %v", err)
	}
	genesisBlock, err := block.Deserialize(genesisBlockBytes)
	if err != nil {
		log.Fatalf("❌ Failed to deserialize genesis block: %v", err)
	}
	fmt.Printf("🔗 Genesis block Merkle root (anchored): %x\n", genesisBlock.ExtraData)

	recoveredTip := network.GetLatestBlockID()
	if bytes.Equal(recoveredTip[:], make([]byte, 32)) {
		// Only set tip to genesis if chain is empty
		if err := network.SetLatestBlockID(genesisBlock.BlockID); err != nil {
			fmt.Printf("[ERROR] Failed to persist latest block ID: %v\n", err)
		}
		fmt.Println("🌟 Tip BlockID loaded into memory:", genesisBlock.BlockID)
		if _, err := store.GetBlock(genesisBlock.BlockID[:]); err != nil {
			fmt.Printf("⚠️  WARNING: Tip BlockID %x is NOT present in LevelDB!\n", genesisBlock.BlockID)
		} else {
			fmt.Println("✅ Tip block is present in LevelDB.")
		}
	} else {
		fmt.Printf("🌟 Tip BlockID loaded from recovery: %x\n", recoveredTip)
	}
	
	// === Block Producer Control ===
	produceBlocks := false
	if os.Getenv("BLOCK_PRODUCER") == "1" {
		produceBlocks = true
	}

	if produceBlocks {
		go func() {
			fmt.Println("[BLOCK PRODUCER] Fallback-enabled goroutine started")
			for {
				// Remove stale/disconnected producers before each round
				network.RemoveDisconnectedProducers()
				// Pretty log the dynamic producer table before each round
				networking.PrintProducerTable(network.ProducersDynamic)
				producers := network.GetSortedDynamicProducers()
				localHeight := network.GetChainHeight()
				localTip := network.GetLatestBlockID()
				// Compute max peer height
				maxPeerHeight := network.MaxPeerHeight()
				fmt.Printf("[DEBUG] Producers: %v\n", producers)
				fmt.Printf("[DEBUG] Local height: %d, Local tip: %x, Max peer height: %d\n", localHeight, localTip, maxPeerHeight)
				//fmt.Println("[DEBUG] Peer table:")
				for _, peer := range network.Peers() {
					fmt.Printf("  - Address: %s, Height: %d, Tip: %s\n", peer.Address, peer.ChainHeight, peer.TipBlockID)
				}
				if localHeight < maxPeerHeight {
					fmt.Println("[SYNC] Not at chain tip, aggressively syncing from peers ahead...")
					for _, peer := range network.Peers() {
						if peer.ChainHeight > localHeight {
							go network.SyncFullChainFromPeer(peer.Address)
						}
					}
					time.Sleep(blockProductionInterval)
					continue
				}
				N := len(producers)
				if N == 0 {
					continue
				}
				height := network.GetChainHeight()
				myKey := fmt.Sprintf("%x", pubKey)
				myIdx := -1
				for i, k := range producers {
					if k == myKey {
						myIdx = i
						break
					}
				}
				leaderIdx := height % N
				fallbackIdx := (leaderIdx + 1) % N

				// --- Consecutive fallback tracking ---
				if myIdx == leaderIdx {
					chain.ConsecutiveFallbacks = 0 // Reset on leader
					// I'm the leader for this slot, try immediately
					fmt.Printf("[BLOCK PRODUCER] My turn (height %d, leader idx %d)\n", height, leaderIdx)
					err := network.ProduceBlock()
					if err != nil {
						fmt.Printf("[BLOCK PRODUCER] Failed: %v\n", err)
					} else {
						fmt.Println("[BLOCK PRODUCER] Block produced.")
					}
					// Wait 1.5s for fallback window
					time.Sleep(blockProductionInterval)
				} else if myIdx == fallbackIdx {
					chain.ConsecutiveFallbacks++
					fmt.Printf("[BLOCK PRODUCER] Fallback turn (height %d, fallback idx %d) [consecutive: %d]\n", height, fallbackIdx, chain.ConsecutiveFallbacks)

					// Wait for grace period before producing fallback
					time.Sleep(fallbackGracePeriod)

					// Check if any peer is ahead after grace period
					maxPeerHeight := network.MaxPeerHeight()
					if localHeight < maxPeerHeight {
						fmt.Println("[FALLBACK] Still behind after grace period, syncing from peers and skipping fallback block this round...")
						for _, peer := range network.Peers() {
							if peer.ChainHeight > localHeight {
								go network.SyncFullChainFromPeer(peer.Address)
							}
						}
						// Skip fallback block production this round
						time.Sleep(blockProductionInterval - fallbackGracePeriod)
						continue
					}

					// If no peer is ahead, proceed with fallback block production
					err := network.ProduceBlock()
					if err != nil {
						fmt.Printf("[BLOCK PRODUCER] Fallback failed: %v\n", err)
					} else {
						fmt.Println("[BLOCK PRODUCER] Fallback block produced.")
					}
					// Wait for next slot
					time.Sleep(blockProductionInterval - fallbackGracePeriod)
				} else {
					chain.ConsecutiveFallbacks = 0 // Reset if not leader or fallback
					// Not my turn, wait for next slot
					time.Sleep(blockProductionInterval)
				}
			}
		}()
	} else {
		fmt.Println("[INFO] Block production is disabled. This node is running in lightweight mode.")
	}

	// === Mempool HTTP Endpoint ===
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/mempool", func(w http.ResponseWriter, r *http.Request) {
			if network.Mempool == nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("No mempool attached"))
				return
			}
			txs := network.Mempool.GetAllTxs()
			ids := make([]string, len(txs))
			for i, tx := range txs {
				ids[i] = tx.TxID
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ids)
		})
		// Listen on a different port to avoid collision with main API (e.g., 8090)
		fmt.Println("[MEMPOOL MONITOR] Serving /mempool on :8090")
		http.ListenAndServe(":8090", mux)
	}()

	// === API Server ===

	// === Setup PeerSet for GossipEngine ===
	peerSet := mempool.NewPeerSet()
	// Add Node B as a peer (this is Node A, so B is on :8082)
	peerSet.AddPeer(mempool.Peer{ID: "node-b", Address: "localhost:8082"})
	gossipEngine := mempool.NewGossipEngine([]string{}, mp)
	gossipEngine.UpdatePeersFromSet(peerSet)
	fmt.Printf("[GOSSIP DEBUG] Peers at startup: %v\n", gossipEngine.Peers)
	forkChoice := chain.NewForkChoice(store)
	// --- Finalizer wiring ---
	finalizerPubKey := os.Getenv("FINALIZER_PUBKEY")
	authorizedFinalizers := []string{}
	if finalizerPubKey != "" {
		authorizedFinalizers = append(authorizedFinalizers, finalizerPubKey)
	}
	// Load and decode private key (never log or print)
	var finalizerPrivKey ed25519.PrivateKey = nil
	keyPath := os.Getenv("FINALIZER_PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = "finalizer_private.key"
	}
	privKeyB64, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Printf("\033[31m[ERROR] Failed to read finalizer_private.key at %s: %v\033[0m\n", keyPath, err)
		os.Exit(1)
	}
	privKeyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(privKeyB64)))
	if err != nil {
		fmt.Printf("\033[31m[ERROR] Failed to base64 decode finalizer_private.key: %v\033[0m\n", err)
		os.Exit(1)
	}
	if len(privKeyBytes) != ed25519.PrivateKeySize {
		fmt.Printf("\033[31m[ERROR] finalizer_private.key is not 64 bytes after base64 decoding (got %d bytes)\033[0m\n", len(privKeyBytes))
		os.Exit(1)
	}
	fmt.Printf("\033[32m[DEBUG] Loaded finalizer private key from %s: length=%d first8=%x last8=%x\033[0m\n", keyPath, len(privKeyBytes), privKeyBytes[:8], privKeyBytes[len(privKeyBytes)-8:])
	finalizerPrivKey = privKeyBytes

	finalizer := block.NewFinalizer(authorizedFinalizers, &FinalizerAuditLogger{}, finalizerPrivKey)
	apiServer := server.NewServer(store, network, apiListenAddr, gossipEngine, forkChoice, finalizer)

	err = apiServer.Start()
	if err != nil {
		log.Fatalf(" Failed to start API server: %v", err)
	}

	// === Keep Alive ===
	select {}
}
