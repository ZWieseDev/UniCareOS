package mempool

import (
	"sync"
	"encoding/json"
	"net/http"
	"bytes"
	"fmt"
)

// GossipEngine manages tx gossip (broadcast/receive)
// Usage: Call UpdatePeersFromSet to sync peers from your PeerSet.
type GossipEngine struct {
	Peers   []string // Peer addresses (host:port)
	SeenTxs map[string]struct{} // Deduplication: TxID -> seen
	Mu      sync.Mutex
	Mempool *Mempool
}

// NewGossipEngine creates a new gossip engine
func NewGossipEngine(peers []string, mempool *Mempool) *GossipEngine {
	return &GossipEngine{
		Peers:   peers,
		SeenTxs: make(map[string]struct{}),
		Mempool: mempool,
	}
}

// UpdatePeersFromSet updates the GossipEngine's peer list from a PeerSet.
func (ge *GossipEngine) UpdatePeersFromSet(ps *PeerSet) {
	peers := ps.ListPeers()
	peerAddresses := make([]string, 0, len(peers))
	for _, p := range peers {
		peerAddresses = append(peerAddresses, p.Address)
	}
	ge.Peers = peerAddresses
}

// BroadcastTx gossips a transaction to all peers
func (ge *GossipEngine) BroadcastTx(tx Transaction) {
	fmt.Printf("[GOSSIP] Broadcasting tx %s to peers: %v\n", tx.TxID, ge.Peers)
	ge.Mu.Lock()
	if _, seen := ge.SeenTxs[tx.TxID]; seen {
		ge.Mu.Unlock()
		return // Already seen
	}
	ge.SeenTxs[tx.TxID] = struct{}{}
	ge.Mu.Unlock()
	msg := GossipMessage{Tx: tx}
	data, _ := json.Marshal(msg)
	fmt.Printf("[GOSSIP] Broadcasting tx %s to peers: %v\n", tx.TxID, ge.Peers)
	for _, peer := range ge.Peers {
		url := fmt.Sprintf("http://%s/gossip_tx", peer)
		fmt.Printf("[GOSSIP] Attempting POST to %s\n", url)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Printf("[GOSSIP] Failed to send tx %s to peer %s: %v\n", tx.TxID, peer, err)
			continue
		}
		resp.Body.Close()
		fmt.Printf("[GOSSIP] Sent tx %s to peer %s\n", tx.TxID, peer)
	}
	ge.Mempool.AddTx(tx)
}

// ReceiveGossip handles an incoming gossip message
func (ge *GossipEngine) ReceiveGossip(data []byte) {
	fmt.Println("[GOSSIP] ReceiveGossip called")
	var msg GossipMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		fmt.Println("[GOSSIP] Received invalid gossip message (unmarshal failed)")
		return // Invalid message
	}
	ge.Mu.Lock()
	if _, seen := ge.SeenTxs[msg.Tx.TxID]; seen {
		ge.Mu.Unlock()
		fmt.Printf("[GOSSIP] Ignored duplicate tx %s\n", msg.Tx.TxID)
		return // Already seen
	}
	ge.SeenTxs[msg.Tx.TxID] = struct{}{}
	ge.Mu.Unlock()
	added := ge.Mempool.AddTx(msg.Tx)
	if added {
		fmt.Printf("[GOSSIP] Added tx %s to mempool\n", msg.Tx.TxID)
	} else {
		fmt.Printf("[GOSSIP] Tx %s was not added (duplicate or mempool full)\n", msg.Tx.TxID)
	}
	// Optionally: re-broadcast to peers
}
