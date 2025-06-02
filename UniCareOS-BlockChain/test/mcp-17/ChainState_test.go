package mcp17

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"unicareos/core/block"
	"unicareos/core/storage"
	"unicareos/core/state"
	"unicareos/core/blockchain"
	"unicareos/types/ids"
)

func TestWriteBlockToState_Integration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := storage.NewStorage(filepath.Join(tmpDir, "leveldb"))
	if err != nil {
		t.Fatal(err)
	}
	st := &state.ChainState{
		StateDB: db,
		Indexes: state.ChainIndexes{
			ByHash:      make(map[string]uint64),
			ByPatientID: make(map[string][]string),
			ByProviderID: make(map[string][]string),
			ByEpoch:     make(map[uint64][]string),
		},
	}

	blockID := ids.ID{1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32}
	eventID := ids.ID{101,102,103,104,105,106,107,108,109,110,111,112,113,114,115,116,117,118,119,120,121,122,123,124,125,126,127,128,129,130,131,132}

	blk := &block.Block{
		BlockID:    blockID,
		MerkleRoot: "root456",
		Height:     1,
		Epoch:      1,
		Events: []block.ChainedEvent{
			{
				EventID:    eventID,
				EventType:  "test",
				Description: "desc",
				Timestamp:   time.Now(),
				PatientID:   "p1",
				ProviderID:  "prov1",
				Epoch:       1,
			},
		},
	}

	receipt, err := state.WriteBlockToState(st, blk, "tester")
	if err != nil || receipt.Status != "committed" {
		t.Fatalf("expected committed, got %v, err %v", receipt.Status, err)
	}

	got, err := st.StateDB.Get("block:" + blk.BlockID.String())
	if err != nil || len(got) == 0 {
		t.Errorf("block not persisted")
	}

	if len(st.Indexes.ByPatientID[blk.Events[0].PatientID]) == 0 {
		t.Errorf("patient index not updated")
	}
	if st.ChainHead != blk.BlockID.String() {
		t.Errorf("chain head not updated")
	}
}

func TestComputeEpochMerkleRoot_Integration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "merkle-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := storage.NewStorage(filepath.Join(tmpDir, "leveldb"))
	if err != nil {
		t.Fatal(err)
	}
	st := &state.ChainState{
		StateDB: db,
		Indexes: state.ChainIndexes{
			ByHash:      make(map[string]uint64),
			ByPatientID: make(map[string][]string),
			ByProviderID: make(map[string][]string),
			ByEpoch:     make(map[uint64][]string),
		},
	}

	chainedEvent := block.ChainedEvent{
		EventID:    ids.ID{201,202,203,204,205,206,207,208,209,210,211,212,213,214,215,216,217,218,219,220,221,222,223,224,225,226,227,228,229,230,231,232},
		EventType:  "finalize_event",
		Description: "finalization",
		Timestamp:   time.Now(),
		PatientID:   "p2",
		ProviderID:  "prov2",
		Epoch:       2,
	}

	blockID := ids.ID{11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42}
	blk := &block.Block{
		BlockID:    blockID,
		MerkleRoot: "dummy_root",
		Height:     1,
		Epoch:      2,
		Events:     []block.ChainedEvent{chainedEvent},
	}

	receipt, err := state.WriteBlockToState(st, blk, "tester")
	if err != nil || receipt.Status != "committed" {
		t.Fatalf("expected committed, got %v, err %v", receipt.Status, err)
	}

	root, err := blockchain.ComputeEpochMerkleRoot(2, db)
	if err != nil {
		t.Fatalf("failed to compute epoch Merkle root: %v", err)
	}
	if root == "" {
		t.Errorf("expected non-empty Merkle root for epoch 2, got empty string")
	}
}