package mempool

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMempoolAddAndEvict(t *testing.T) {
	mp := NewMempool(2)
	tx1 := Transaction{TxID: "tx1", Timestamp: time.Now().Unix()}
	tx2 := Transaction{TxID: "tx2", Timestamp: time.Now().Unix()}
	tx3 := Transaction{TxID: "tx3", Timestamp: time.Now().Unix()}

	if !mp.AddTx(tx1) {
		t.Fatal("failed to add tx1")
	}
	if !mp.AddTx(tx2) {
		t.Fatal("failed to add tx2")
	}
	if !mp.AddTx(tx3) {
		t.Fatal("failed to add tx3 (should evict tx1)")
	}
	if _, ok := mp.GetTx("tx1"); ok {
		t.Error("tx1 should have been evicted")
	}
	if _, ok := mp.GetTx("tx2"); !ok {
		t.Error("tx2 should be present")
	}
	if _, ok := mp.GetTx("tx3"); !ok {
		t.Error("tx3 should be present")
	}
}

func TestGossipDeduplication(t *testing.T) {
	mp := NewMempool(10)
	ge := NewGossipEngine([]string{"peer1", "peer2"}, mp)

tx := Transaction{TxID: "tx42", Timestamp: time.Now().Unix()}
	// Simulate broadcasting
	ge.BroadcastTx(tx)
	if _, ok := ge.SeenTxs[tx.TxID]; !ok {
		t.Error("tx should be marked as seen after broadcast")
	}
	// Simulate receiving the same tx (should be ignored)
	msg := GossipMessage{Tx: tx}
	data, _ := json.Marshal(msg)
	ge.ReceiveGossip(data)
	// Only one instance in mempool
	count := 0
	for _, tx := range mp.GetAllTxs() {
		if tx.TxID == "tx42" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 tx42 in mempool, got %d", count)
	}
}

func TestPeerSet(t *testing.T) {
	ps := NewPeerSet()
	p1 := Peer{ID: "peerA", Address: "127.0.0.1:1234"}
	p2 := Peer{ID: "peerB", Address: "127.0.0.1:5678"}
	ps.AddPeer(p1)
	ps.AddPeer(p2)
	if len(ps.ListPeers()) != 2 {
		t.Error("expected 2 peers in set")
	}
	ps.RemovePeer("peerA")
	if _, ok := ps.GetPeer("peerA"); ok {
		t.Error("peerA should have been removed")
	}
}
