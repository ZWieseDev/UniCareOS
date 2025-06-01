package mempool

import (
    "sync"
    "time"
)

// Mempool manages pending transactions for gossip and block inclusion
type Mempool struct {
	mu    sync.Mutex
	txs   map[string]Transaction // TxID -> Transaction
	order []string              // FIFO order for eviction
	maxTxs int                  // Max transactions in pool
	ExpiredPool *ExpiredTxPool  // Archive for expired transactions
}

// NewMempool creates a new mempool with a maximum size
func NewMempool(maxTxs int) *Mempool {
	return &Mempool{
		txs:    make(map[string]Transaction),
		order:  make([]string, 0),
		maxTxs: maxTxs,
		ExpiredPool: NewExpiredTxPool(),
	}
} 

// AddTx adds a transaction to the pool (returns false if duplicate or at capacity)
func (mp *Mempool) AddTx(tx Transaction) bool {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	if _, exists := mp.txs[tx.TxID]; exists {
		return false // duplicate
	}
	if len(mp.txs) >= mp.maxTxs {
		// Evict oldest
		oldest := mp.order[0]
		delete(mp.txs, oldest)
		mp.order = mp.order[1:]
	}
	mp.txs[tx.TxID] = tx
	mp.order = append(mp.order, tx.TxID)
	return true
}

// RemoveTx removes a transaction by TxID
func (mp *Mempool) RemoveTx(txID string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	if _, exists := mp.txs[txID]; exists {
		delete(mp.txs, txID)
		for i, id := range mp.order {
			if id == txID {
				mp.order = append(mp.order[:i], mp.order[i+1:]...)
				break
			}
		}
	}
}

// GetTx returns a transaction by TxID (and bool for existence)
func (mp *Mempool) GetTx(txID string) (Transaction, bool) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	tx, ok := mp.txs[txID]
	return tx, ok
}

// GetAllTxs returns all transactions in the pool
func (mp *Mempool) GetAllTxs() []Transaction {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	txs := make([]Transaction, 0, len(mp.txs))
	for _, id := range mp.order {
		txs = append(txs, mp.txs[id])
	}
	return txs
}

// PurgeExpired moves transactions older than a given duration to the ExpiredTxPool
func (mp *Mempool) PurgeExpired(maxAge time.Duration) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	now := time.Now().Unix()
	newOrder := make([]string, 0, len(mp.order))
	for _, id := range mp.order {
		tx := mp.txs[id]
		if now-tx.Timestamp > int64(maxAge.Seconds()) {
			// Archive the expired transaction
			if mp.ExpiredPool != nil {
				existing, ok := mp.ExpiredPool.GetExpiredTx(id)
				if ok {
					// Only update ExpiredAt and Reason, preserve ResubmitCount and LastError
					existing.ExpiredAt = time.Now()
					existing.Reason = "timeout"
					mp.ExpiredPool.AddExpiredTx(existing)
				} else {
					empTx := ExpiredTx{
						TxID: id,
						Payload: tx.Payload,
						ExpiredAt: time.Now(),
						Reason: "timeout",
					}
					mp.ExpiredPool.AddExpiredTx(empTx)
				}
			}
			delete(mp.txs, id)
		} else {
			newOrder = append(newOrder, id)
		}
	}
	mp.order = newOrder
}
