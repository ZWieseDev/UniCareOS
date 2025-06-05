package networking

import (
	"bufio"
	"unicareos/core/state"
	"bytes"
	"encoding/json"
	"fmt"

	"sort"
	"io"
	"runtime/debug"
	"net"
	"net/http"
	"sync"
	"time"
	"strings"
	"encoding/hex"
	"encoding/binary"

	"unicareos/core/block"
	"unicareos/core/storage"
	"unicareos/core"
	"unicareos/core/mempool"
	"unicareos/core/chain"
	"unicareos/core/blockchain"
	
	)


type Peer struct {
	Address     string    // host:port
	APIPort     int
	ChainHeight int
	TipBlockID  string
	LastSeen    time.Time
	HostOnly    string    // host only, for broadcast URL
	PubKey      []byte    // Ed25519 public key (added for producer table)
}


type Network struct {
	ChainState *state.ChainState // Pointer to ChainState for epoch tracking
	// ... existing fields ...
	BlockProductionInterval time.Duration
	// Dynamic set of block producers (pubkey hex → present)
	ProducersDynamic map[string]struct{}
	// Missed turn counters (pubkey hex → count)
	MissedTurns map[string]int
	// Orphan block buffer: BlockID hex → block.Block
	OrphanBlocks map[string]block.Block
	// ... existing fields ...
	// [REMOVED] Producers [][]byte and ProducerIndex are now obsolete; use ProducersDynamic and GetSortedDynamicProducers()
// Producers [][]byte // List of block producer public keys (raw bytes)
// ProducerIndex int  // This node's index in the producers list (or -1 if not a producer)

	listenAddr    string
	apiPort       int
	peers         []Peer
	store         *storage.Storage
	lock          sync.Mutex
	latestBlockID [32]byte

	EpochBlockCount int // Number of blocks per epoch, from genesis config

	recentBlocks      map[string]struct{} // BlockID hex → exists (for deduplication)
	bannedPeers       map[string]time.Time
	peerRequestCounts map[string][]time.Time
	banCounts         map[string]int // Tracks number of bans per IP for progressive banning

	PrivKey []byte // Ed25519 private key
	PubKey  []byte // Ed25519 public key

	Mempool *mempool.Mempool // Reference to the mempool for block production
}

func NewNetwork(listenAddr string, store *storage.Storage, apiPort int, pubKey, privKey []byte, chainState *state.ChainState, epochBlockCount int) *Network {
	// Cleanup: Remove any address-based entries from ProducersDynamic
	cleanupProducerTable := func(producers map[string]struct{}) {
		for k := range producers {
			// Ed25519 public keys are 32 bytes (64 hex chars)
			if len(k) != 64 {
				delete(producers, k)
				fmt.Printf("[CLEANUP] Removed non-pubkey entry from ProducersDynamic: %s\n", k)
			}
		}
	}

	n := &Network{
		ChainState: chainState,
		EpochBlockCount: epochBlockCount,
		listenAddr:    listenAddr,
		apiPort:       apiPort,
		peers:         []Peer{},
		store:         store,
		latestBlockID: [32]byte{},
		recentBlocks:  make(map[string]struct{}),
		bannedPeers:   make(map[string]time.Time),
		peerRequestCounts: make(map[string][]time.Time),
		banCounts:     make(map[string]int),
		PrivKey:       privKey,
		PubKey:        pubKey,

		ProducersDynamic: make(map[string]struct{}),
		MissedTurns:      make(map[string]int),
	}
	cleanupProducerTable(n.ProducersDynamic)
	// Always ensure own pubkey is present
	n.AddProducer(pubKey)
	// Dynamic producer table only (no static config):
	// Already added own pubkey above, dynamic set will be populated via handshakes.

	// (Removed for production: node starts unbanned by default)
	// n.BanPeer("127.0.0.1", 10*time.Minute)
	// n.BanPeer("127.0.0.2", 10*time.Minute)
	if n.store != nil && n.store.DB() != nil {
		n.LoadBanState()
	}
	return n
}

// LoadBanState loads persistent bans and ban counts from LevelDB
func (n *Network) LoadBanState() {
	imported := 0
	iter := n.store.DB().NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		if len(key) > 4 && string(key[:4]) == "ban:" {
			address := string(key[4:])
			expiryStr := string(iter.Value())
			expiry, err := time.Parse(time.RFC3339, expiryStr)
			if err == nil {
				n.bannedPeers[address] = expiry
				imported++
			}
		}
		if len(key) > 9 && string(key[:9]) == "banCount:" {
			address := string(key[9:])
			if len(iter.Value()) == 8 {
				n.banCounts[address] = int(binary.BigEndian.Uint64(iter.Value()))
			}
		}
	}
	iter.Release()
	fmt.Printf("[BAN LOAD] Imported %d persistent bans from DB\n", imported)
}


// RecoverTipFromStorage scans all blocks and sets the tip to the block at the end of the main chain
func (n *Network) RecoverTipFromStorage() {
	blockMap := make(map[string]block.Block)
	parentMap := make(map[string]string)
	ids, err := n.store.ListBlockIDs()
	if err != nil {
		fmt.Println("[RECOVER] Could not list block IDs:", err)
		return
	}
	fmt.Println("[RECOVER] All block IDs in DB:")
	for _, idHex := range ids {
		idStr := string(idHex)
		blockID, err := hex.DecodeString(idStr)
		if err != nil {
			fmt.Printf("  [RECOVER] Could not decode blockID hex %s: %v\n", idStr, err)
			continue
		}
		blkBytes, err := n.store.GetBlock(blockID)
		if err != nil {
			fmt.Printf("  [RECOVER] Could not load block %s: %v\n", idStr, err)
			continue
		}
		blkPtr, err := block.Deserialize(blkBytes)
		if err != nil {
			fmt.Printf("  [RECOVER] Could not deserialize block %s: %v\n", idStr, err)
			continue
		}
		blk := *blkPtr
		fmt.Printf("  BlockID: %s, PrevHash: %s, Height: %d\n", idStr, blk.PrevHash, blk.Height)
		blockMap[idStr] = blk
		if blk.PrevHash != "" {
			parentMap[blk.PrevHash] = idStr
		}
	}
	fmt.Println("[RECOVER] blockMap (BlockID -> Height):")
	for id, blk := range blockMap {
		fmt.Printf("  %s -> %d\n", id, blk.Height)
	}
	fmt.Println("[RECOVER] parentMap (PrevHash -> BlockID):")
	for prev, id := range parentMap {
		fmt.Printf("  %s -> %s\n", prev, id)
	}
	// Find tip: block that is not referenced as a parent by any other block
	fmt.Println("[RECOVER] Candidate tips:")
	tipID := ""
	for id := range blockMap {
		if _, ok := parentMap[id]; !ok {
			fmt.Printf("  Candidate tip: %s (height %d)\n", id, blockMap[id].Height)
			if tipID == "" || blockMap[id].Height > blockMap[tipID].Height {
				tipID = id
			}
		}
	}
	if tipID != "" {
		var tipIDArr [32]byte
		decoded, err := hex.DecodeString(tipID)
		if err == nil && len(decoded) == 32 {
			copy(tipIDArr[:], decoded)
			n.lock.Lock()
			n.latestBlockID = tipIDArr
			n.lock.Unlock()
			// Persist recovered tip to LevelDB
			err = n.store.DB().Put([]byte("latestBlockID"), tipIDArr[:], nil)
			if err != nil {
				fmt.Printf("[RECOVER] Failed to persist recovered tip: %v\n", err)
			} else {
				fmt.Printf("[RECOVER] Tip recovered as block %s (persisted)\n", tipID)
			}
		}
	} else {
		fmt.Println("[RECOVER] No tip found!")
	}
}



// --- Producer Table Debug Helper ---
func PrintProducerTable(producers map[string]struct{}) {
	fmt.Println("[PRODUCER TABLE] Current dynamic producers:")
	for k := range producers {
		fmt.Printf("  - %s\n", k)
	}
}

// SetLatestBlockID sets the latest block ID in a thread-safe manner and persists it to the DB.
func (n *Network) SetLatestBlockID(id [32]byte) error {
    n.lock.Lock()
    defer n.lock.Unlock()
    n.latestBlockID = id
    if n.store != nil && n.store.DB() != nil {
        if err := n.store.DB().Put([]byte("latestBlockID"), id[:], nil); err != nil {
            return err
        }
    }
    return nil
}

// --- P2P TCP Communication ---

// ProduceBlock creates and broadcasts a new block with the current tip as parent, using the dynamic producer table.
func (n *Network) ProduceBlock() error {
	// RED DEBUG PRINT: Confirm block producer code is running

	producers := n.GetSortedDynamicProducers()
	if len(producers) == 0 {
		return fmt.Errorf("no dynamic producers available")
	}
	height := n.getChainHeight()
	myKey := fmt.Sprintf("%x", n.PubKey)
	myIdx := -1
	for i, k := range producers {
		if k == myKey {
			myIdx = i
			break
		}
	}
	if myIdx == -1 {
		return fmt.Errorf("this node's pubkey not in dynamic producer table")
	}
	leaderIdx := height % len(producers)
	//fmt.Printf("[PRODUCE DEBUG] Height: %d\n", height)
	//fmt.Printf("[PRODUCE DEBUG] Producers: %v\n", producers)
	//fmt.Printf("[PRODUCE DEBUG] MyIdx: %d, LeaderIdx: %d\n", myIdx, leaderIdx)
	//fmt.Printf("[PRODUCE DEBUG] MyKey: %s, LeaderKey: %s\n", myKey, producers[leaderIdx])
	if myIdx != leaderIdx {
		fmt.Printf("[LEADER] Not my turn (height %d, leader idx %d, my idx %d)\n", height, leaderIdx, myIdx)
		return nil // Not this node's turn
	}


	n.lock.Lock()
	defer n.lock.Unlock()

	tipID := n.latestBlockID
	var parentHeight uint64
	var parentHash string
	if !bytes.Equal(tipID[:], make([]byte, 32)) {
		blkBytes, err := n.store.GetBlock(tipID[:])
		if err != nil {
			return fmt.Errorf("could not fetch parent block: %v", err)
		}
		blkPtr, err := block.Deserialize(blkBytes)
		if err != nil {
			return fmt.Errorf("could not deserialize parent block: %v", err)
		}
		parentHeight = blkPtr.Height
		parentHash = fmt.Sprintf("%x", blkPtr.BlockID[:])
	} else {
		parentHeight = 0
		parentHash = ""
	}

	nextHeight := parentHeight + 1

	// Gather transactions from the mempool (deterministic ordering)
	var events []block.ChainedEvent
	var includedTxIDs []string
	// Create newBlock *before* processing transactions so we can pass its pointer
	newBlock := block.Block{
		Version:         "",
		ProtocolVersion: "",
		Height:          nextHeight,
		PrevHash:        parentHash,
		MerkleRoot:      "",
		Timestamp:       time.Now(),
		Events:          nil, // Will fill after processing
		BanEvents:       nil, // TODO: gather pending ban events
		ExtraData:       nil,
		ValidatorDID:    fmt.Sprintf("ed25519:%x", n.PubKey), // Store public key as DID
	}
	if n.Mempool != nil {
		txs := n.Mempool.GetAllTxs()
		for _, tx := range txs {
			// Attempt to interpret each transaction as a medical record submission
			var submission block.MedicalRecordSubmission
			err := json.Unmarshal(tx.Payload, &submission)
			if err != nil {

				continue // Skip invalid submissions
			}
			// Call SubmitRecordToBlock to process, validate, and append event
				// --- Full-chain lineage lookup (moved from block package) ---
				docLineage := []string{}
				if submission.RevisionOf != "" {
					found := false
					// Search current block
					for _, evt := range newBlock.Events {
						if evt.EventID.String() == submission.RevisionOf {
							if evt.DocLineage != nil {
								docLineage = append(docLineage, evt.DocLineage...)
							}
							docLineage = append(docLineage, submission.RevisionOf)
							found = true
							break
						}
					}
					// If not found, search previous blocks recursively
					if !found && n.store != nil {
						prevEventID := submission.RevisionOf
						currentHeight := int(newBlock.Height) - 1
						// Traverse all the way back to the genesis block. This prevents infinite loops as the search will always terminate.
for currentHeight >= 0 && len(prevEventID) > 0 {
							blk, err := n.store.GetBlockByHeight(currentHeight)
							if err != nil {
								break // out of blocks
							}
							fmt.Printf("[LINEAGE DEBUG] Block height %d: numEvents=%d\n", currentHeight, len(blk.Events))
							for _, e := range blk.Events {
								fmt.Printf("[LINEAGE DEBUG]   eventID=%s\n", e.EventID.String())
							}

							foundEvt := false
							for _, evt := range blk.Events {
								fmt.Printf("[LINEAGE DEBUG] Comparing evt.EventID.String() = '%s' to prevEventID = '%s'\n", evt.EventID.String(), prevEventID)
fmt.Printf("[LINEAGE DEBUG] evt.EventID.Bytes = %x\n", evt.EventID[:])
fmt.Printf("[LINEAGE DEBUG] prevEventID (from JSON) = %s\n", prevEventID)
// If you have a hex decode utility, decode prevEventID to bytes and print
if prevBytes, err := hex.DecodeString(prevEventID); err == nil {
    fmt.Printf("[LINEAGE DEBUG] prevEventID.Bytes = %x\n", prevBytes)
} else {
    fmt.Printf("[LINEAGE DEBUG] prevEventID hex decode error: %v\n", err)
}
								if evt.EventID.String() == prevEventID {
									if evt.DocLineage != nil {
										docLineage = append(docLineage, evt.DocLineage...)
									}
									docLineage = append(docLineage, prevEventID)
									prevEventID = evt.RevisionOf
									foundEvt = true
									break
								}
							}
							if !foundEvt {
    // Not found in this block, keep searching older blocks
    currentHeight--
    continue
}
currentHeight--
							currentHeight--
						}
						// Debug: print full constructed lineage
						//fmt.Printf("[LINEAGE] Full revision lineage for %s: %v\n", submission.RevisionOf, docLineage)
					}
				}
				// Reverse lineage to chronological order (oldest first)
				for i, j := 0, len(docLineage)-1; i < j; i, j = i+1, j-1 {
					docLineage[i], docLineage[j] = docLineage[j], docLineage[i]
				}
				//fmt.Printf("[LINEAGE] Final lineage for event: %v\n", docLineage)
				submission.DocLineage = docLineage
				_, err = block.SubmitRecordToBlock(submission, &newBlock)
			if err != nil {

				continue // Skip failed submissions
			}
			// Find the event just appended (last in newBlock.Events)
			if len(newBlock.Events) > 0 {
				evt := newBlock.Events[len(newBlock.Events)-1]
				events = append(events, evt)
				includedTxIDs = append(includedTxIDs, tx.TxID)
				// Optionally: print event info

				// Print the lineage actually written to the event in the block
				//fmt.Printf("[LINEAGE DEBUG] Event %s written to block with lineage: %v\n", evt.EventID.String(), evt.DocLineage)
			}
		}
	}
	// After processing, assign events to newBlock
	newBlock.Events = events
	// --- Set block epoch based on block height and epoch block count ---
	epochBlockCount := uint64(n.EpochBlockCount)
	if epochBlockCount == 0 {
		epochBlockCount = 1 // prevent division by zero
	}
	if nextHeight > 0 {
		newBlock.Epoch = (nextHeight - 1) / epochBlockCount
		fmt.Printf("[EPOCH] Setting block epoch to %d for block at height %d (epochBlockCount=%d)\n", newBlock.Epoch, newBlock.Height, epochBlockCount)
	} else {
		newBlock.Epoch = 0
		fmt.Printf("[EPOCH] Setting block epoch to 0 for genesis block\n")
	}

	if len(includedTxIDs) > 0 {

	}

	// Compute BlockID first (for header hash)
	newBlock.BlockID = newBlock.ComputeID()
	// Sign the block header with Ed25519
	if len(n.PrivKey) == 64 {
		newBlock.Signature = core.Sign(n.PrivKey, newBlock.BlockID[:])
	} else {
		fmt.Println("[WARN] No valid Ed25519 private key loaded; block will not be signed!")
	}

	//fmt.Printf("[DEBUG] About to compute BlockID for Height %d, PrevHash %s\n", newBlock.Height, newBlock.PrevHash)
	//fmt.Printf("[DEBUG] Computed BlockID: %x\n", newBlock.BlockID[:])
if len(newBlock.Signature) > 0 {
	//fmt.Printf("[DEBUG] Block Signature: %x\n", newBlock.Signature)
} else {
	//fmt.Println("[DEBUG] Block Signature: (none)")
}
	blkBytes, err := newBlock.Serialize()
	if err != nil {
		return fmt.Errorf("could not serialize new block: %v", err)
	}

	err = n.store.SaveBlock(newBlock.BlockID[:], blkBytes)
	if err != nil {
		return fmt.Errorf("could not save new block: %v", err)
	}
	err = n.store.DB().Put([]byte("latestBlockID"), newBlock.BlockID[:], nil)
	if err != nil {
		return fmt.Errorf("could not update latestBlockID: %v", err)
	}
	n.latestBlockID = newBlock.BlockID

	// Remove included transactions from the mempool
	if n.Mempool != nil && len(includedTxIDs) > 0 {
		for _, txID := range includedTxIDs {
			n.Mempool.RemoveTx(txID)
		}

	}
	fmt.Printf("[CHAIN] Block produced at height %d (BlockID: %x)\n", newBlock.Height, newBlock.BlockID[:])

	// --- Epoch Finalization Enhancement for Local Block Production ---
	if n.ChainState != nil {
		n.ChainState.BlocksInEpoch++
		if n.ChainState.BlocksInEpoch >= uint64(n.EpochBlockCount) {
			epochNumber := n.ChainState.Epoch
			// Compute Merkle root for the epoch
			epochSummaryHash, err := blockchain.ComputeEpochMerkleRoot(epochNumber, n.store)
			if err != nil {
				fmt.Println("[EPOCH] Failed to compute epoch Merkle root:", err)
			} else {
				// Sign the epoch summary hash with the block producer's key
				finalizerSignature := ""
				if len(n.PrivKey) == 64 {
					sigBytes := core.Sign(n.PrivKey, []byte(epochSummaryHash))
					finalizerSignature = hex.EncodeToString(sigBytes)
				}
				auditLogID := "" // TODO: wire in real audit log ID
				_, receipt, err := blockchain.FinalizeEpoch(n.store, n.ChainState, epochNumber, finalizerSignature, auditLogID)
				if err != nil {
					fmt.Println("[EPOCH] FinalizeEpoch failed:", err)
				} else {
					fmt.Printf("\033[33m[EPOCH FINALIZED] Epoch %d finalized. Receipt: %+v\033[0m\n", epochNumber, receipt)
				}
				n.ChainState.Epoch++
				n.ChainState.BlocksInEpoch = 0
			}
		}
		err := n.ChainState.SaveEpochState()
		if err != nil {
			fmt.Printf("[EPOCH] Failed to persist epoch state: %v\n", err)
		}
	}

	blkIDHex := fmt.Sprintf("%x", newBlock.BlockID[:])
	// --- Compact propagation: announce block header first ---
	n.BroadcastBlockAnnouncement(blkIDHex, newBlock.Height, newBlock.PrevHash, newBlock.Timestamp.Unix())
	// --- Optionally: short delay to let peers request block (not required) ---
	// time.Sleep(100 * time.Millisecond)

	// --- Fallback: still broadcast full block for backward compatibility ---
	n.BroadcastNewBlock(blkBytes, blkIDHex)


	return nil
}




func (n *Network) Start() error {
	ln, err := net.Listen("tcp", n.listenAddr)
	if err != nil {
		return err
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			go n.handleConnection(conn)
		}
	}()


	return nil
}

// handleConnection handles an incoming connection from a peer
func (n *Network) handleConnection(conn net.Conn) {
	defer func() {
		// On connection close, clean up disconnected producers
		n.RemoveDisconnectedProducers()
		conn.Close()
	}()
	defer conn.Close()

	// --- Ban enforcement for incoming P2P connections (by IP only) ---
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		host = conn.RemoteAddr().String()
	}
	if n.IsPeerBanned(host) {
		fmt.Printf("[BAN] Rejected incoming P2P connection from banned peer: %s\n", host)
		return
	}

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	reader := bufio.NewReader(conn)

	line, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Println(" Read error from peer:", err)
		return
	}

	var peerHello Peer
	err = json.Unmarshal(line, &peerHello)
	if err != nil {
		fmt.Println(" Peer hello parse error:", err)
		return
	}
	// Patch HostOnly if missing (for backward compatibility)
	if peerHello.HostOnly == "" {
		host, _, _ := net.SplitHostPort(peerHello.Address)
		peerHello.HostOnly = host
	}
	fmt.Printf("[DEBUG] Received peerHello: Addr=%s, APIPort=%d, ChainHeight=%d, HostOnly=%s\n", peerHello.Address, peerHello.APIPort, peerHello.ChainHeight, peerHello.HostOnly)

	// === DYNAMIC PRODUCER TABLE: Add peer (if pubkey available) ===
	// TODO: Extract and use peer's public key when available
	// Example: n.AddProducer(peerPubKey)
	// For now, use peer address as a placeholder (remove when switching to pubkey)
	if len(peerHello.PubKey) > 0 {
		n.AddProducer(peerHello.PubKey)
	} else {
		fmt.Printf("[WARNING] Peer hello missing public key, cannot add to producer table: %s\n", peerHello.Address)
	}
	n.lock.Lock()
	// Store/update peer info under the address we connected to
	// Normalize to full canonical address (host:port)
// Always use peerHello.Address (host) and peerHello.APIPort for canonical address
// Print raw incoming address for debug
fmt.Printf("[DEBUG] Raw incoming peerHello.Address: '%s'\n", peerHello.Address)
// Always use the remote address's host part, ignore what the peer sent!
remoteHost, remotePort, _ := net.SplitHostPort(conn.RemoteAddr().String())
address := net.JoinHostPort(remoteHost, remotePort) // Use P2P port!
fmt.Printf("[DEBUG] handleConnection: Forced peer address: '%s' (P2P), API Port: %d\n", address, peerHello.APIPort)
peerHello.Address = address // Always set to canonical host:port
found := false
for i, p := range n.peers {
	if p.Address == address {
		n.peers[i].Address = address // Ensure canonical address is always stored
		n.peers[i].APIPort = peerHello.APIPort
		n.peers[i].ChainHeight = peerHello.ChainHeight
		n.peers[i].TipBlockID = peerHello.TipBlockID
		n.peers[i].LastSeen = peerHello.LastSeen
		found = true
		break
	}
}
if !found {
	n.peers = append(n.peers, Peer{
		Address:     address,
		APIPort:     peerHello.APIPort,
		ChainHeight: peerHello.ChainHeight,
		TipBlockID:  peerHello.TipBlockID,
		LastSeen:    peerHello.LastSeen,
	})
}
fmt.Printf("[DEBUG] Peer table after update:\n")
for _, p := range n.peers {
	fmt.Printf("    - P2P Address: '%s', API Port: %d, Height: %d, Tip: %s\n", p.Address, p.APIPort, p.ChainHeight, p.TipBlockID)
}
n.lock.Unlock();

	// === DYNAMIC PRODUCER TABLE: Add peer (if pubkey available) ===
	// TODO: Extract and use peer's public key when available
	// Example: n.AddProducer(peerPubKey)
	// For now, use peer address as a placeholder (remove when switching to pubkey)
	if len(peerHello.PubKey) > 0 {
		n.AddProducer(peerHello.PubKey)
	} else {
		fmt.Printf("[WARNING] Peer hello missing public key, cannot add to producer table: %s\n", peerHello.Address)
	}

	// --- Always trigger sync logic, let it decide ---
	fmt.Printf("[SYNC DECISION] Triggering sync logic for peer %s\n", address)
	go n.SyncFullChainFromPeer(address)


	// --- Send our own hello back for two-way handshake ---
	myTipID := n.GetLatestBlockID()
	myHello := Peer{
		Address:     n.listenAddr,
		APIPort:     n.apiPort,
		ChainHeight: n.getChainHeight(),
		TipBlockID:  fmt.Sprintf("%x", myTipID[:]),
		LastSeen:    time.Now().UTC(),
		PubKey:      n.PubKey, // include our public key in handshake
	}
	fmt.Println("[HANDSHAKE] After sending hello, dynamic producer table:")
	PrintProducerTable(n.ProducersDynamic)
	myHelloBytes, err := json.Marshal(myHello)
	if err == nil {
		_, _ = conn.Write(append(myHelloBytes, '\n'))
		fmt.Printf("[DEBUG] Sent hello in response: Addr=%s, APIPort=%d\n", myHello.Address, myHello.APIPort)
	}

	fmt.Printf(" New peer connected: %+v\n", peerHello)
}

// ConnectToPeer connects to a peer on the network
func (n *Network) ConnectToPeer(address string) error {
	defer func() {
		// After connection attempt, clean up disconnected producers
		n.RemoveDisconnectedProducers()
	}()
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("tcp dial failed: %w", err)
	}
	defer conn.Close()

	tipID := n.GetLatestBlockID()

	// Step 1: Send JSON hello over TCP
	host, _, _ := net.SplitHostPort(n.listenAddr)
	hello := Peer{
		Address:     n.listenAddr,
		APIPort:     n.apiPort, // <--- ensure this is correct
		ChainHeight: n.getChainHeight(),
		TipBlockID:  fmt.Sprintf("%x", tipID[:]),
		LastSeen:    time.Now().UTC(),
		HostOnly:    host,
		PubKey:      n.PubKey, // include our public key in handshake
	}
	fmt.Printf("[DEBUG] Sending hello: Addr=%s, APIPort=%d\n", hello.Address, hello.APIPort)

	helloBytes, err := json.Marshal(hello)
	if err != nil {
		return fmt.Errorf("marshal hello failed: %w", err)
	}

	_, err = conn.Write(append(helloBytes, '\n'))
	if err != nil {
		return fmt.Errorf("send hello failed: %w", err)
	}

	// --- Add two-way handshake: wait for peer's hello back ---
	reader := bufio.NewReader(conn)
	peerHelloLine, err := reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("failed to read peer hello: %w", err)
	}
	var peerHello Peer
	err = json.Unmarshal(peerHelloLine, &peerHello)
	if err != nil {
		return fmt.Errorf("failed to parse peer hello: %w", err)
	}
	fmt.Printf("[DEBUG] Received peerHello from connect: Addr=%s, APIPort=%d, ChainHeight=%d\n", peerHello.Address, peerHello.APIPort, peerHello.ChainHeight)

	// === DYNAMIC PRODUCER TABLE: Add peer (if pubkey available) ===
	// TODO: Extract and use peer's public key when available
	// Example: n.AddProducer(peerPubKey)
	// For now, use peer address as a placeholder (remove when switching to pubkey)
	if len(peerHello.PubKey) > 0 {
		n.AddProducer(peerHello.PubKey)
	} else {
		fmt.Printf("[WARNING] Peer hello missing public key, cannot add to producer table: %s\n", peerHello.Address)
	}

	// Add or update peer table with peerHello
    // Use the actual address we connected to, not what the peer claims!
    remoteHost, remotePort, _ := net.SplitHostPort(conn.RemoteAddr().String())
    canonicalAddr := net.JoinHostPort(remoteHost, remotePort) // Use P2P port!
    peerHello.Address = canonicalAddr

    n.lock.Lock()
    found := false
    for i, p := range n.peers {
        if p.Address == canonicalAddr {
            n.peers[i] = peerHello
            found = true
            break
        }
    }
    if !found {
        n.peers = append(n.peers, peerHello)
    }
    n.lock.Unlock()

	// Step 2: Launch async sync check (via HTTP)
	go func(peerAddr string) {
		// Always use peer's API port for chain tip and block requests
		fmt.Println("[DEBUG] Peer table before getPeerAPIPort in ConnectToPeer chain tip call:")
		printPeerTable(n.peers)
		fmt.Printf("[DEBUG] Looking up API port for: %s\n", peerAddr)
		apiPort := n.getPeerAPIPort(peerAddr)
		fmt.Printf("[DEBUG] Got API port: %d\n", apiPort)
		host, _, err := net.SplitHostPort(peerAddr)
		if err != nil {
			host = peerAddr
		}
		// Chain tip URL (if needed)
		tipURL := fmt.Sprintf("http://%s:%d/get_chain_tip", host, apiPort)
		fmt.Printf("[DEBUG] get_chain_tip URL: %s\n", tipURL)
		// Request block URL
		requestBlockURL := fmt.Sprintf("http://%s:%d/request_block", host, apiPort)
		fmt.Printf("[DEBUG] request_block URL: %s\n", requestBlockURL)

		// Example: fetch block (replace with actual logic as needed)
		payload := map[string]string{
			"blockID": fmt.Sprintf("%x", tipID[:]),
		}
		body, _ := json.Marshal(payload)
		resp, err := http.Post(requestBlockURL, "application/json", bytes.NewReader(body))
		if err != nil {
			fmt.Printf("Could not fetch chain tip from %s: %v\n", peerAddr, err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Could not fetch chain tip from %s: %s\n", peerAddr, resp.Status)
			return
		}
		blockBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("[SYNC_BLOCK] Block as string: %s\n", string(blockBytes))

		blkPtr, err := block.Deserialize(blockBytes)
		if err != nil {
			fmt.Printf("❌ Failed to deserialize peer block from %s: %v\n", peerAddr, err)
			return
		}
		blk := *blkPtr

		// Save block
		err = n.store.SaveBlock(blk.BlockID[:], blockBytes)
		if err != nil {
			fmt.Printf("❌ Failed to save block from %s: %v\n", peerAddr, err)
			return
		}

		// Update tip
		n.lock.Lock()
n.latestBlockID = blk.BlockID
n.lock.Unlock()
if n.store != nil && n.store.DB() != nil {
    err := n.store.DB().Put([]byte("latestBlockID"), blk.BlockID[:], nil)
    if err != nil {
        fmt.Printf("[ERROR] Failed to persist latest block ID: %v\n", err)
    }
}
		fmt.Printf("✅ Synced block %x from %s\n", blk.BlockID[:], peerAddr)
	}(address)

	return nil
}



// --- Memory Chaining Logic ---

// SyncFullChainFromPeer fetches all missing blocks from a peer until fully synced
// Always normalize to host:port for lookup
func (n *Network) getPeerHeightFromTable(address string) int {
	fmt.Printf("[DEBUG] getPeerHeightFromTable: looking up address '%s'\n", address)
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = "127.0.0.1"
		port = address
	}
	canonical := net.JoinHostPort(host, port)
	fmt.Printf("[DEBUG] getPeerHeightFromTable: canonical address '%s'\n", canonical)

	n.lock.Lock()
	defer n.lock.Unlock()
	fmt.Printf("[DEBUG] Peer table at lookup:\n")
	for _, p := range n.peers {
		fmt.Printf("    - Address: '%s', APIPort: %d, Height: %d, Tip: %s\n", p.Address, p.APIPort, p.ChainHeight, p.TipBlockID)
	}

	// Remove duplicate declaration, use assignment
	host, port, err = net.SplitHostPort(address)
	if err != nil {
		host = "127.0.0.1"
		port = address
	}
	canonical = net.JoinHostPort(host, port)

	for _, p := range n.peers {
		fmt.Printf("[DEBUG] getPeerHeightFromTable: comparing against peer table entry '%s' (height=%d)\n", p.Address, p.ChainHeight)
		if p.Address == canonical {
			fmt.Printf("[DEBUG] getPeerHeightFromTable: FOUND, returning height %d\n", p.ChainHeight)
			return p.ChainHeight
		}
	}
	fmt.Printf("[DEBUG] getPeerHeightFromTable: NOT FOUND, returning 0\n")
	return 0
}

// SyncFullChainFromPeer fetches all missing blocks from a peer until fully synced, with rate limiting and only if peer is ahead
func (n *Network) SyncFullChainFromPeer(address string) error {
	myHeight := n.getChainHeight()
	peerHeight := n.getPeerHeightFromTable(address)
	fmt.Printf("[DEBUG] After getPeerHeightFromTable: peerHeight=%d\n", peerHeight)
	fmt.Printf("[SYNC] My height: %d, Peer height: %d\n", myHeight, peerHeight)
	// Only sync if peer is ahead
	if peerHeight <= myHeight {
		fmt.Println("[SYNC] Peer is not ahead, skipping sync.")
		return nil
	}
	// Rate limit: avoid repeated syncs within short window
	if !n.shouldSyncNow(address) {
		fmt.Println("[SYNC] Rate limit: skipping sync for peer.")
		return nil
	}
	for h := myHeight; h < peerHeight; h++ {
		blockID, err := n.getPeerBlockIDByHeight(address, h)
		if err != nil {
			fmt.Printf("❌ [SYNC ERROR] Failed to get blockID for height %d: %v\n", h, err)
			return fmt.Errorf("sync aborted: failed to get blockID for height %d: %v", h, err)
		}
		blockBytes, err := n.RequestBlockFromPeer(address, blockID)
		if err != nil {
			fmt.Printf("❌ [SYNC ERROR] Failed to fetch block at height %d: %v\n", h, err)
			return fmt.Errorf("sync aborted: failed to fetch block at height %d: %v", h, err)
		}
		blkPtr, err := block.Deserialize(blockBytes)
		if err != nil {
			fmt.Printf("❌ [SYNC ERROR] Failed to deserialize block at height %d: %v\n", h, err)
			return fmt.Errorf("sync aborted: failed to deserialize block at height %d: %v", h, err)
		}
		err = n.store.SaveBlock(blkPtr.BlockID[:], blockBytes)
		if err != nil {
			fmt.Printf("❌ [SYNC ERROR] Failed to save block at height %d: %v\n", h, err)
			return fmt.Errorf("sync aborted: failed to save block at height %d: %v", h, err)
		}
		n.lock.Lock()
n.latestBlockID = blkPtr.BlockID
n.lock.Unlock()
if n.store != nil && n.store.DB() != nil {
    err := n.store.DB().Put([]byte("latestBlockID"), blkPtr.BlockID[:], nil)
    if err != nil {
        fmt.Printf("[ERROR] Failed to persist latest block ID: %v\n", err)
    }
}
		fmt.Printf("✅ Synced block %x at height %d from peer\n", blkPtr.BlockID[:], h)
	}

	// After sync, compare tips
	myTip := n.GetLatestBlockID()
	peerTip, err := n.getPeerBlockIDByHeight(address, peerHeight-1)
	if err != nil {
		fmt.Printf("[SYNC WARNING] Could not fetch peer tip blockID for comparison: %v\n", err)
	} else {
		if myTip != peerTip {
			fmt.Printf("[SYNC WARNING] After sync, tip mismatch! Local tip: %x, Peer tip: %x\n", myTip[:], peerTip[:])
		} else {
			fmt.Printf("[SYNC] After sync, tips match: %x\n", myTip[:])
		}
	}
	return nil
}

// getPeerChainHeight fetches the chain height from a peer's /chain_height endpoint
func (n *Network) getPeerChainHeight(address string) (int, error) {
	fmt.Println("[DEBUG] Peer table before getPeerAPIPort in getPeerChainHeight:")
printPeerTable(n.peers)
fmt.Printf("[DEBUG] Looking up API port for: %s\n", address)
apiPort := n.getPeerAPIPort(address)
fmt.Printf("[DEBUG] Got API port: %d\n", apiPort)
host, _, err := net.SplitHostPort(address)
if err != nil {
    host = address
}
fmt.Printf("[DEBUG] getPeerChainHeight: host=%s, apiPort=%d\n", host, apiPort)
url := fmt.Sprintf("http://%s:%d/chain_height", host, apiPort)
fmt.Printf("[DEBUG] getPeerChainHeight URL: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to get peer chain height: %v", err)
	}
	defer resp.Body.Close()
	var data struct { Height int `json:"height"` }
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return 0, fmt.Errorf("failed to decode peer chain height: %v", err)
	}
	return data.Height, nil
}

// getPeerBlockIDByHeight fetches the block ID at a given height from a peer's /blocks endpoint
func (n *Network) getPeerBlockIDByHeight(address string, height int) ([32]byte, error) {
	apiPort := n.getPeerAPIPort(address)
host, _, err := net.SplitHostPort(address)
if err != nil {
    host = address
}
url := fmt.Sprintf("http://%s:%d/blocks?start=%d&end=%d", host, apiPort, height, height)
fmt.Printf("[DEBUG] getPeerBlockIDByHeight URL: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get peer block by height: %v", err)
	}
	defer resp.Body.Close()
	var blocks []struct { BlockID [32]byte `json:"block_id"` }
	err = json.NewDecoder(resp.Body).Decode(&blocks)
	if err != nil || len(blocks) == 0 {
		return [32]byte{}, fmt.Errorf("no block found at height %d", height)
	}
	return blocks[0].BlockID, nil
}

func (n *Network) GetLatestBlockID() [32]byte {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.latestBlockID
}

// BlockAnnounceMessage is used to announce new blocks (compact propagation)
type BlockAnnounceMessage struct {
	BlockID    string `json:"block_id"`
	Height     uint64 `json:"height"`
	PrevHash   string `json:"prev_hash"`
	Timestamp  int64  `json:"timestamp"`
}

// BlockBroadcastMessage is used for P2P block propagation
// (keep simple for now, can add signature later)
type BlockBroadcastMessage struct {
	BlockBytes []byte `json:"block_bytes"`
	BlockID    string `json:"block_id"` // hex string for deduplication
}

// reclaimAndDiscardOrphanBlock reclaims all transactions from an orphan block to the mempool and deletes the orphan from storage.
func (n *Network) reclaimAndDiscardOrphanBlock(blk block.Block) {
    // Reclaim all events/txs to mempool
    for _, event := range blk.Events {
        // Use EventID as the unique TxID for mempool.Transaction
        tx := mempool.Transaction{
            TxID:   event.EventID.String(),
            // You can serialize the event as needed for Payload
            Payload:   nil, // Optionally: serialize event if needed
            Timestamp: event.Timestamp.Unix(),
            Sender:    event.AuthorValidator.String(),
        }
        n.Mempool.AddTx(tx)
    }
    // Remove from orphan buffer if present
    orphanID := blk.BlockID.String()
    n.lock.Lock()
    if n.OrphanBlocks != nil {
        delete(n.OrphanBlocks, orphanID)
    }
    n.lock.Unlock()
    // Delete from storage
    if n.store != nil {
        err := n.store.DeleteBlock(blk.BlockID[:])
        if err != nil {
            fmt.Printf("[ORPHAN] Failed to delete orphan block %x from storage: %v\n", blk.BlockID[:], err)
        } else {
            fmt.Printf("[ORPHAN] Orphan block %x deleted from storage.\n", blk.BlockID[:])
        }
    }
}

func (n *Network) SaveNewBlock(blk block.Block) error {
	fmt.Println("[DEBUG] Entered SaveNewBlock for block height:", blk.Height, "blockID:", blk.BlockID)

	// --- Epoch tracking ---
	if n.ChainState != nil {
		// Increment BlocksInEpoch
		n.ChainState.BlocksInEpoch++
		fmt.Println("[DEBUG] Checking for epoch boundary: BlocksInEpoch=", n.ChainState.BlocksInEpoch, "EpochBlockCount=", n.EpochBlockCount)
		// Get EpochBlockCount (assume field for now)
		if n.ChainState.BlocksInEpoch >= uint64(n.EpochBlockCount) { // epoch boundary
			// Epoch-end logic: finalize the epoch
			finalizerSignature := "" // TODO: wire in real signature
			auditLogID := "" // TODO: wire in real audit log ID
			epochNumber := n.ChainState.Epoch
			_, receipt, err := blockchain.FinalizeEpoch(n.store, n.ChainState, epochNumber, finalizerSignature, auditLogID)
			if err != nil {
				fmt.Println("[EPOCH] FinalizeEpoch failed:", err)
			} else {
				fmt.Printf("\033[33m[EPOCH FINALIZED] Epoch %d finalized. Receipt: %+v\033[0m\n", epochNumber, receipt)
			}
			// Advance epoch
			n.ChainState.Epoch++
			n.ChainState.BlocksInEpoch = 0
		}
		// Persist epoch state
		err := n.ChainState.SaveEpochState()
		if err != nil {
			fmt.Printf("[EPOCH] Failed to persist epoch state: %v\n", err)
		}
	}

    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("[PANIC] SaveNewBlock panicked: %v\n", r)
            debug.PrintStack()
        }
    }()

    // --- Apply all BanEvents in this block to the local ban list ---
    n.lock.Lock()
    for _, ban := range blk.BanEvents {
        expiry, err := ban.ExpiryTime()
        if err == nil {
            n.bannedPeers[ban.Address] = expiry
        }
    }
    n.lock.Unlock()

    // Strict tip check and update under a single lock
    n.lock.Lock()
    defer n.lock.Unlock()

    currentTipHex := fmt.Sprintf("%x", n.latestBlockID[:])
    isGenesis := blk.PrevHash == "" || blk.PrevHash == strings.Repeat("0", len(blk.PrevHash))
    fmt.Printf("[DEBUG] Before block acceptance: currentTip=%s, incoming.PrevHash=%s, incoming.BlockID=%s\n", currentTipHex, blk.PrevHash, blk.BlockID)

    if blk.BlockID == n.latestBlockID {
        fmt.Printf("Block %x is already the tip. Skipping save.\n", blk.BlockID[:])
        return nil
    }

    if !isGenesis && blk.PrevHash != currentTipHex {
		fmt.Printf("[CHAIN] Block accepted at height %d (BlockID: %x)\n", blk.Height, blk.BlockID[:])
        chain.ConsecutiveFallbacks = 0
        fmt.Println("[FALLBACK] Reset fallback counter after accepting new block")
    } else {
        fmt.Printf("[ORPHAN] Block %x is an orphan (PrevHash %s does not match current tip %s). Discarding and reclaiming transactions.\n", blk.BlockID[:], blk.PrevHash, currentTipHex)
        n.reclaimAndDiscardOrphanBlock(blk)
        fmt.Println("[DEBUG] After orphan discard, before fork-choice reorg")
        go n.HandleForkChoiceReorg("", 0, "") // non-blocking fork-choice reorg
        return nil
    }
    return nil
}

// RefreshPeerHeights actively queries all peers for their latest chain heights
func (n *Network) RefreshPeerHeights() {
    peers := n.Peers()
    for _, peer := range peers {
        height, err := n.getPeerChainHeight(peer.Address)
        if err != nil {
            fmt.Printf("[PEER REFRESH] Failed to query peer %s: %v\n", peer.Address, err)
            continue
        }
        n.lock.Lock()
        peer.ChainHeight = height
        n.lock.Unlock()
        fmt.Printf("[PEER REFRESH] Peer %s height updated to %d\n", peer.Address, height)
    }
}


func (n *Network) getChainHeight() int {
	tipID := n.GetLatestBlockID()
	if bytes.Equal(tipID[:], make([]byte, 32)) {
		return 0 // No blocks
	}
	blkBytes, err := n.store.GetBlock(tipID[:])
	if err != nil {
		return 0
	}
	blkPtr, err := block.Deserialize(blkBytes)
	if err != nil {
		return 0
	}
	return int(blkPtr.Height)
}

// ✅ New: CheckPeerTips
func (n *Network) CheckPeerTips() []map[string]interface{} {
	n.lock.Lock()
	peers := make([]Peer, len(n.peers))
	copy(peers, n.peers)
	n.lock.Unlock()

	report := []map[string]interface{}{} // ✅ force empty array, not nil
	myHeight := n.getChainHeight()

	tipID := n.GetLatestBlockID()
	myTip := fmt.Sprintf("%x", tipID[:])

	for _, peer := range peers {
		status := "ok"

		if peer.ChainHeight > myHeight {
			status = "peer ahead (needs sync)"
		} else if peer.TipBlockID != myTip {
			status = "tip mismatch"
		}

		report = append(report, map[string]interface{}{
			"peerAddress":  peer.Address,
			"peerHeight":   peer.ChainHeight,
			"peerTipBlock": peer.TipBlockID,
			"myHeight":     myHeight,
			"myTipBlock":   myTip,
			"status":       status,
		})
	}

	return report
}


func (n *Network) getPeerAPIPort(address string) int {
	n.lock.Lock()
	defer n.lock.Unlock()
	for _, p := range n.peers {
		if p.Address == address {
			fmt.Printf("[DEBUG] getPeerAPIPort: %s → %d\n", address, p.APIPort)
			return p.APIPort
		}
	}
	fmt.Printf("[DEBUG] getPeerAPIPort: %s → fallback 8080\n", address)
	return 8080 // fallback
}

func (n *Network) RequestBlockFromPeer(address string, blockID [32]byte) ([]byte, error) {
	fmt.Printf("[DEBUG][FETCH] Requesting parent block %x from peer %s\n", blockID[:], address)
	fmt.Println("[DEBUG] Peer table before getPeerAPIPort in RequestBlockFromPeer:")
printPeerTable(n.peers)
fmt.Printf("[DEBUG] Looking up API port for: %s\n", address)
apiPort := n.getPeerAPIPort(address)
fmt.Printf("[DEBUG] Got API port: %d\n", apiPort)
host, _, err := net.SplitHostPort(address)
if err != nil {
    host = address // fallback if no port
}
url := fmt.Sprintf("http://%s:%d/request_block?block_id=%x", host, apiPort, blockID[:])
fmt.Printf("[DEBUG][FETCH] RequestBlockFromPeer URL: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("[DEBUG][FETCH] HTTP request to %s failed: %v\n", url, err)
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		fmt.Printf("[DEBUG][FETCH] Peer %s responded with error: %s – %s\n", address, resp.Status, string(data))
		return nil, fmt.Errorf("peer error: %s – %s", resp.Status, string(data))
	}
	fmt.Printf("[DEBUG][FETCH] Peer %s responded with block data for %x\n", address, blockID[:])
	return io.ReadAll(resp.Body)
}



func (n *Network) SyncFromPeer(address string, knownTipID [32]byte) error {
	fmt.Println("🔄 Syncing from peer:", address)
	if bytes.Equal(knownTipID[:], make([]byte, 32)) {
		fmt.Println("⚠️ Aborting sync: tip is zero.")
		return nil
	}
	
	blockBytes, err := n.RequestBlockFromPeer(address, knownTipID)
	if err != nil {
		return fmt.Errorf("❌ Failed to fetch block from peer: %v", err)
	}

	blkPtr, err := block.Deserialize(blockBytes)
	if err != nil {
		return fmt.Errorf("❌ Failed to deserialize peer block: %v", err)
	}
	blk := *blkPtr

	err = n.store.SaveBlock(blk.BlockID[:], blockBytes)
	if err != nil {
		return fmt.Errorf("❌ Failed to save block: %v", err)
	}

	n.lock.Lock()
	defer n.lock.Unlock()
	currentTip := n.latestBlockID
	currentTipHex := fmt.Sprintf("%x", currentTip[:])
	isGenesis := blk.PrevHash == "" || blk.PrevHash == strings.Repeat("0", len(blk.PrevHash))
	if isGenesis || blk.PrevHash == currentTipHex {
		err = n.store.DB().Put([]byte("latestBlockID"), blk.BlockID[:], nil)
		if err != nil {
			return fmt.Errorf("❌ Failed to update latestBlockID: %v", err)
		}
		n.latestBlockID = blk.BlockID
		fmt.Printf("✅ Synced block %x from peer (tip updated)\n", blk.BlockID[:])
	} else {
		fmt.Printf("⚠️ Block %x from peer is an orphan (PrevHash %s does not match current tip %s). Tip not updated.\n", blk.BlockID[:], blk.PrevHash, currentTipHex)
	}
	return nil
}
// BroadcastBlockAnnouncement sends a block header/announce to all peers
func (n *Network) BroadcastBlockAnnouncement(blockIDHex string, height uint64, prevHash string, timestamp int64) {
	msg := BlockAnnounceMessage{
		BlockID:   blockIDHex,
		Height:    height,
		PrevHash:  prevHash,
		Timestamp: timestamp,
	}
	data, _ := json.Marshal(msg)
	for _, peer := range n.peers {
		go func(p Peer) {
			host := p.HostOnly
			if host == "" {
				host, _, _ = net.SplitHostPort(p.Address)
			}
			url := fmt.Sprintf("http://%s:%d/announce_block", host, p.APIPort)
			_, err := http.Post(url, "application/json", bytes.NewReader(data))
			if err != nil {
				fmt.Printf("[ANNOUNCE] Error sending block announce to %s: %v\n", url, err)
			}
		}(peer)
	}
}

// BroadcastNewBlock sends a new block to all peers via HTTP POST
func (n *Network) BroadcastNewBlock(blockBytes []byte, blockIDHex string) {
	msg := BlockBroadcastMessage{
		BlockBytes: blockBytes,
		BlockID:    blockIDHex,
	}
	data, _ := json.Marshal(msg)
	for _, peer := range n.peers {
		go func(p Peer) {
			host := p.HostOnly
			if host == "" {
				host, _, _ = net.SplitHostPort(p.Address)
			}
			url := fmt.Sprintf("http://%s:%d/broadcast_block", host, p.APIPort)
			_, err := http.Post(url, "application/json", bytes.NewReader(data))
			if err != nil {
				fmt.Printf("[BROADCAST] Error sending block to %s: %v\n", url, err)
			}
		}(peer)
	}
}

// HandleAnnounceBlock processes an incoming block announcement (header only)
func (n *Network) HandleAnnounceBlock(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if n.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		fmt.Printf("[BAN] Blocked announce from banned peer: %s\n", host)
		return
	}
	if !n.AllowPeerRequest(host) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		fmt.Printf("[RATE LIMIT] Blocked announce from %s\n", host)
		return
	}
	var msg BlockAnnounceMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// If we already have the block, ignore
	n.lock.Lock()
	if _, exists := n.recentBlocks[msg.BlockID]; exists {
		n.lock.Unlock()
		w.WriteHeader(http.StatusOK)
		return
	}
	n.lock.Unlock()
	// Otherwise, request the full block from the announcing peer
	go func() {
		blockIDBytes, err := hex.DecodeString(msg.BlockID)
		if err != nil || len(blockIDBytes) != 32 {
			fmt.Printf("[ANNOUNCE] Invalid block ID in announce: %s\n", msg.BlockID)
			return
		}
		var blockID [32]byte
		copy(blockID[:], blockIDBytes)
		// Try to fetch full block from announcer
		peerAddr := host
		blockBytes, err := n.RequestBlockFromPeer(peerAddr, blockID)
		if err != nil {
			fmt.Printf("[ANNOUNCE] Failed to fetch announced block %s from %s: %v\n", msg.BlockID, peerAddr, err)
			return
		}
		blkPtr, err := block.Deserialize(blockBytes)
		if err != nil {
			fmt.Printf("[ANNOUNCE] Failed to deserialize announced block %s: %v\n", msg.BlockID, err)
			return
		}
		err = n.SaveNewBlock(*blkPtr)
		if err != nil {
			fmt.Printf("[ANNOUNCE] Could not save announced block %s: %v\n", msg.BlockID, err)
			return
		}
		fmt.Printf("[ANNOUNCE] Successfully fetched and saved announced block %s\n", msg.BlockID)
fmt.Printf("[LOG] Current tip after announce: %x\n", n.GetLatestBlockID())
	}()
	w.WriteHeader(http.StatusOK)
}

// HandleBroadcastBlock processes an incoming block broadcast from a peer
func (n *Network) HandleBroadcastBlock(w http.ResponseWriter, r *http.Request) {
	// --- Ban & Rate Limit Enforcement (by IP only) ---
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr // fallback if parsing fails
	}
	if n.IsPeerBanned(host) {
		http.Error(w, "forbidden: banned", http.StatusForbidden)
		fmt.Printf("[BAN] Blocked request from banned peer: %s\n", host)
		return
	}
	if !n.AllowPeerRequest(host) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		fmt.Printf("[RATE LIMIT] Blocked request from %s\n", host)
		return
	}
	// NOTE: For best practice, apply this logic to all peer-facing handlers.

	var msg BlockBroadcastMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// Deduplication
	n.lock.Lock()
	if _, exists := n.recentBlocks[msg.BlockID]; exists {
		n.lock.Unlock()
		w.WriteHeader(http.StatusOK)
		return
	}
	n.recentBlocks[msg.BlockID] = struct{}{}
	n.lock.Unlock()
	// Deserialize and validate block
	blkPtr, err := block.Deserialize(msg.BlockBytes)
	if err != nil {
		http.Error(w, "invalid block", http.StatusBadRequest)
		return
	}
	blk := *blkPtr
	// === Save Block
	err = n.SaveNewBlock(blk)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not save block: %v", err), http.StatusBadRequest)
		return
	}
	// Relay: rebroadcast to other peers
	n.BroadcastNewBlock(msg.BlockBytes, msg.BlockID)
	w.WriteHeader(http.StatusOK)
}

// [REMOVED] Static producer config loader. All producer logic is now dynamic and handshake-based.

// AddProducer adds a pubkey (as bytes) to the dynamic producer set
func (n *Network) AddProducer(pubkey []byte) {
	hexKey := fmt.Sprintf("%x", pubkey)
	n.ProducersDynamic[hexKey] = struct{}{}
	fmt.Printf("[PRODUCER] Added producer %s\n", hexKey)
	PrintProducerTable(n.ProducersDynamic)
}

// RemoveProducer removes a pubkey (as bytes) from the dynamic producer set and missed-turns
func (n *Network) RemoveProducer(pubkey []byte) {
	hexKey := fmt.Sprintf("%x", pubkey)
	delete(n.ProducersDynamic, hexKey)
	delete(n.MissedTurns, hexKey)
	fmt.Printf("[PRODUCER] Removed producer %s\n", hexKey)
	PrintProducerTable(n.ProducersDynamic)
}

// RemoveDisconnectedProducers scans the dynamic producer table and removes any producer whose pubkey is not present in the peer table (excluding our own pubkey).
func (n *Network) RemoveDisconnectedProducers() {
	n.lock.Lock()
	defer n.lock.Unlock()
	present := make(map[string]struct{})
	now := time.Now()
	for _, p := range n.peers {
		// Only keep peers seen within 2 block intervals
		if now.Sub(p.LastSeen) < 2*n.BlockProductionInterval && len(p.PubKey) == 32 {
			present[fmt.Sprintf("%x", p.PubKey)] = struct{}{}
		}
	}
	// Always keep our own pubkey
	ownKey := fmt.Sprintf("%x", n.PubKey)
	present[ownKey] = struct{}{}
	for key := range n.ProducersDynamic {
		if _, ok := present[key]; !ok {
			fmt.Printf("[PRODUCER] Removing disconnected/stale producer %s\n", key)
			delete(n.ProducersDynamic, key)
			delete(n.MissedTurns, key)
		}
	}
	PrintProducerTable(n.ProducersDynamic)
}

// GetSortedDynamicProducers returns a sorted slice of pubkey hex strings from the dynamic producer table
func (n *Network) GetSortedDynamicProducers() []string {
	n.lock.Lock()
	defer n.lock.Unlock()
	producers := make([]string, 0, len(n.ProducersDynamic))
	for k := range n.ProducersDynamic {
		producers = append(producers, k)
	}
	sort.Strings(producers)
	return producers
}


// TODO: On peer disconnect, call n.RemoveProducer(peerPubKey)

// --- Sync Rate Limiting Helper ---
var lastSyncTimes = make(map[string]time.Time)

func (n *Network) shouldSyncNow(address string) bool {
	now := time.Now()
	last, exists := lastSyncTimes[address]
	if exists && now.Sub(last) < 1*time.Second {
		return false
	}
	lastSyncTimes[address] = now
	return true
}

// GetChainHeight returns the local chain height (number of blocks)
func (n *Network) GetChainHeight() int {
	return n.getChainHeight()
}

// Print the peer table for debugging
func printPeerTable(peers []Peer) {
    fmt.Println("[PEERTABLE] Known peers:")
    for _, p := range peers {
        fmt.Printf("  - Address: %s, APIPort: %d, Height: %d, Tip: %s\n", p.Address, p.APIPort, p.ChainHeight, p.TipBlockID)
    }
}

// Peers returns a copy of the current peer list (thread-safe)
func (n *Network) Peers() []Peer {
    n.lock.Lock()
    defer n.lock.Unlock()
    peersCopy := make([]Peer, len(n.peers))
    copy(peersCopy, n.peers)
    return peersCopy
}

// TriggerSyncIfBehind checks if any peer is ahead and triggers a sync if needed (rate-limited).
func (n *Network) TriggerSyncIfBehind() {
    // Actively refresh peer heights before checking
    n.RefreshPeerHeights()
    
    peers := n.Peers()
    myHeight := n.getChainHeight()
    fmt.Printf("[DEBUG][SYNC] My height: %d\n", myHeight)
    for _, p := range peers {
        fmt.Printf("[DEBUG][SYNC] Peer %s height: %d\n", p.Address, p.ChainHeight)
    }
    bestPeer := ""
    bestHeight := myHeight
    for _, p := range peers {
        if p.ChainHeight > bestHeight {
            bestHeight = p.ChainHeight
            bestPeer = p.Address
        }
    }
    if bestPeer != "" && bestHeight > myHeight && n.shouldSyncNow(bestPeer) {
        fmt.Printf("[SYNC] Triggered by orphan: peer %s is ahead (height %d), syncing...\n", bestPeer, bestHeight)
        err := n.SyncFullChainFromPeer(bestPeer)
        if err != nil {
            fmt.Printf("[SYNC] Sync failed: %v\n", err)
        } else {
            fmt.Printf("[SYNC] Sync complete.\n")
        }
    }
}

// MaxPeerHeight returns the maximum chain height among all peers
func (n *Network) MaxPeerHeight() int {
    n.lock.Lock()
    defer n.lock.Unlock()
    max := 0
    now := time.Now()
    // Only consider peers seen in the last 2 block intervals
    for _, peer := range n.peers {
        if now.Sub(peer.LastSeen) < 2*n.BlockProductionInterval {
            if peer.ChainHeight > max {
                max = peer.ChainHeight
            }
        }
    }
    return max
}