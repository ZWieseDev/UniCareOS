package mempool

import "sync"

// Peer represents a gossip peer (can be extended with richer metadata)
type Peer struct {
	ID      string // Unique peer ID (e.g., pubkey or address)
	Address string // Network address (host:port)
	// Add more fields as needed (e.g., last seen, reputation, etc.)
}

// PeerSet manages a set of peers for the gossip layer
type PeerSet struct {
	mu    sync.Mutex
	peers map[string]Peer // ID -> Peer
}

// NewPeerSet creates a new empty PeerSet
func NewPeerSet() *PeerSet {
	return &PeerSet{
		peers: make(map[string]Peer),
	}
}

// AddPeer adds or updates a peer
func (ps *PeerSet) AddPeer(peer Peer) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.peers[peer.ID] = peer
}

// RemovePeer removes a peer by ID
func (ps *PeerSet) RemovePeer(id string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.peers, id)
}

// GetPeer returns a peer by ID (and bool for existence)
func (ps *PeerSet) GetPeer(id string) (Peer, bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	peer, ok := ps.peers[id]
	return peer, ok
}

// ListPeers returns a slice of all peers
func (ps *PeerSet) ListPeers() []Peer {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	peers := make([]Peer, 0, len(ps.peers))
	for _, peer := range ps.peers {
		peers = append(peers, peer)
	}
	return peers
}
