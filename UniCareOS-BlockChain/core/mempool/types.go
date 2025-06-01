package mempool

// Transaction represents a mempool transaction (simplified for illustration)
type Transaction struct {
	TxID      string // Unique transaction hash
	Payload   []byte // Serialized transaction payload
	Timestamp int64  // Unix timestamp
	Sender    string // (optional) sender address or pubkey
}

// GossipMessage represents a transaction gossip message
type GossipMessage struct {
	Tx Transaction
	// Optionally, add fields for protocol version, signature, etc.
}
