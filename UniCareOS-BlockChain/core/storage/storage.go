package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"bytes"
	"time"
	"strings"
	"encoding/hex"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
    "unicareos/core/block"
)
// StateBackend abstracts the persistent key-value store for blockchain state.
type StateBackend interface {
	Get(key string) ([]byte, error)
	Put(key string, value []byte) error
}





type Storage struct {
	db *leveldb.DB
}

// Get retrieves a value by key from LevelDB.
func (s *Storage) Get(key string) ([]byte, error) {
	return s.db.Get([]byte(key), nil)
}

// Put stores a key-value pair in LevelDB.
func (s *Storage) Put(key string, value []byte) error {
	return s.db.Put([]byte(key), value, nil)
}

func NewStorage(path string) (*Storage, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &Storage{db: db}, nil
}

func (s *Storage) SaveBlock(blockID []byte, blockData []byte) error {
	enc, err := Encrypt(blockData)
	if err != nil {
		return err
	}
	// Unmarshal to get block height
	var blk block.Block
	err = json.Unmarshal(blockData, &blk)
	if err != nil {
		return err
	}
	blockKey := []byte("block:" + fmt.Sprintf("%x", blockID))
	heightKey := []byte(fmt.Sprintf("height:%d", blk.Height))
	batch := new(leveldb.Batch)
	batch.Put(blockKey, enc)
	batch.Put(heightKey, blockID)
	return s.db.Write(batch, nil)
}

func (s *Storage) GetBlock(blockID []byte) ([]byte, error) {
	blockKey := []byte("block:" + fmt.Sprintf("%x", blockID))
	enc, err := s.db.Get(blockKey, nil)
	if err != nil {
		return nil, err
	}
	return Decrypt(enc)
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) HasGenesisBlock() (bool, error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		if bytes.HasPrefix(key, []byte("block:")) {
			return true, nil // Found at least one block
		}
	}
	return false, nil
}

// ✅ NEW: GetLatestBlockID returns the latest block ID in DB.
func (s *Storage) GetLatestBlockID() ([32]byte, error) {
	var latestID [32]byte
	data, err := s.db.Get([]byte("latestBlockID"), nil)
	if err != nil {
		return latestID, err
	}
	copy(latestID[:], data)
	return latestID, nil
}



// GetBlockIDByHeight returns the blockID for a given height
func (s *Storage) GetBlockIDByHeight(height int) ([]byte, error) {
	heightKey := []byte(fmt.Sprintf("height:%d", height))
	blockID, err := s.db.Get(heightKey, nil)
	if err != nil {
		return nil, errors.New("blockID not found for height")
	}
	return blockID, nil
}

// GetBlockByHeight uses the height index for O(1) lookup
func (s *Storage) GetBlockByHeight(height int) (block.Block, error) {
	var blk block.Block
	blockID, err := s.GetBlockIDByHeight(height)
	if err != nil {
		return blk, err
	}
	data, err := s.GetBlock(blockID)
	if err != nil {
		return blk, err
	}
	err = json.Unmarshal(data, &blk)
	return blk, err
}
func (s *Storage) Iterator() iterator.Iterator {
	return s.db.NewIterator(nil, nil)
}

func (s *Storage) GetGenesisBlock() ([]byte, error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if bytes.HasPrefix(key, []byte("block:")) {
			dec, err := Decrypt(iter.Value())
			if err != nil {
				return nil, err
			}
			return dec, nil
		}
	}
	return nil, fmt.Errorf("no genesis block found")
}

func (s *Storage) ListRecentBlocks(max int) ([]map[string]string, error) {
	var summaries []map[string]string

	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	count := 0
	for iter.Last(); iter.Valid() && count < max; iter.Prev() {
		key := iter.Key()
		if !bytes.HasPrefix(key, []byte("block:")) {
			continue // Only process actual block data
		}

		var blk block.Block
		dec, err := Decrypt(iter.Value())
		if err != nil {
			continue // skip broken blocks or decryption errors
		}
		err = json.Unmarshal(dec, &blk)
		if err != nil {
			continue // skip broken blocks
		}

		summaries = append(summaries, map[string]string{
			"blockID":   fmt.Sprintf("%x", blk.BlockID[:]),
			"prevHash":  blk.PrevHash,
			"timestamp": blk.Timestamp.UTC().Format(time.RFC3339),
		})

		count++
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return summaries, nil
}

// RollbackToBlock rolls back the chain to the given block ID (fork point), deleting all blocks after it.
func (s *Storage) RollbackToBlock(forkPoint [32]byte) error {
	// 1. Build a set of blockIDs to keep (from genesis to forkPoint)
	keep := map[string]bool{}
	current := forkPoint
	for {
		blkBytes, err := s.GetBlock(current[:])
		if err != nil {
			break // Stop at missing block
		}
		var blk block.Block
		err = json.Unmarshal(blkBytes, &blk)
		if err != nil {
			break
		}
		keep[fmt.Sprintf("%x", current[:])] = true
		if blk.PrevHash == "" || blk.PrevHash == strings.Repeat("0", len(blk.PrevHash)) {
			break // Reached genesis
		}
		prev, err := hex.DecodeString(blk.PrevHash)
		if err != nil || len(prev) != 32 {
			break
		}
		copy(current[:], prev)
	}

	// 2. Iterate over all blocks, delete those not in keep
	ids, err := s.ListBlockIDs()
	if err != nil {
		return err
	}
	for _, idHex := range ids {
		idStr := string(idHex)
		if !keep[idStr] {
			blockKey := []byte("block:" + idStr)
			height := -1
			blkBytes, err := s.GetBlock([]byte(idHex))
			if err == nil {
				var blk block.Block
				err = json.Unmarshal(blkBytes, &blk)
				if err == nil {
					height = int(blk.Height)
				}
			}
			s.db.Delete(blockKey, nil)
			if height >= 0 {
				heightKey := []byte(fmt.Sprintf("height:%d", height))
				s.db.Delete(heightKey, nil)
			}
		}
	}
	// 3. Update latestBlockID to forkPoint
	err = s.db.Put([]byte("latestBlockID"), forkPoint[:], nil)
	if err != nil {
		return err
	}
	return nil
}

// File: core/storage/storage.go

func (s *Storage) GetChainHeight() (int, error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	height := 0
	for iter.Next() {
		key := iter.Key()
		if bytes.HasPrefix(key, []byte("block:")) {
			height++
		}
	}

	if err := iter.Error(); err != nil {
		return 0, err
	}

	return height, nil
}
// DB exposes the underlying LevelDB instance
func (s *Storage) DB() *leveldb.DB {
	return s.db
}
func (s *Storage) ListBlockIDs() ([][]byte, error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	var ids [][]byte
	for iter.Next() {
		key := iter.Key()
		if !bytes.HasPrefix(key, []byte("block:")) {
			continue
		}
		// Remove the "block:" prefix for returned IDs
		idHex := key[len("block:"):]
		ids = append(ids, append([]byte{}, idHex...))
	}
	return ids, iter.Error()
}
func (s *Storage) DeleteBlock(id []byte) error {
	return s.db.Delete(id, nil)
}
