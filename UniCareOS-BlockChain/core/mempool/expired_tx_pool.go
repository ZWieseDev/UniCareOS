package mempool

import (
	"sync"
	"time"
)

// ExpiredTx represents a transaction that has expired from the mempool.
type ExpiredTx struct {
	TxID         string
	Payload      interface{} // The original transaction payload (can be MedicalRecordSubmission or similar)
	ExpiredAt    time.Time
	Reason       string      // e.g., "timeout"
	ResubmitCount int
	ResubmissionTxIDs []string
	LastError   string // Last failure reason for smarter error handling
}

// ExpiredTxPool is a thread-safe in-memory storage for expired transactions.
type ExpiredTxPool struct {
	pool map[string]ExpiredTx
	lock sync.RWMutex
}

func NewExpiredTxPool() *ExpiredTxPool {
	return &ExpiredTxPool{
		pool: make(map[string]ExpiredTx),
	}
}

// AddExpiredTx adds an expired transaction to the pool.
func (e *ExpiredTxPool) AddExpiredTx(tx ExpiredTx) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.pool[tx.TxID] = tx
}

// GetExpiredTx retrieves an expired transaction by TxID.
func (e *ExpiredTxPool) GetExpiredTx(txID string) (ExpiredTx, bool) {
	e.lock.RLock()
	defer e.lock.RUnlock()
	tx, ok := e.pool[txID]
	return tx, ok
}

// ListExpiredTxs returns all expired transactions.
func (e *ExpiredTxPool) ListExpiredTxs() []ExpiredTx {
	e.lock.RLock()
	defer e.lock.RUnlock()
	txs := make([]ExpiredTx, 0, len(e.pool))
	for _, tx := range e.pool {
		txs = append(txs, tx)
	}
	return txs
}
