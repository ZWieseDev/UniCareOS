package scripts

import (
	"fmt"
	"encoding/json"
	"sort"
	"unicareos/core/storage"
	"unicareos/core/block"
)

func ScanChain() {
	dbPath := "./unicareos_db"

	store, err := storage.NewStorage(dbPath)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	iter := store.Iterator()
	defer iter.Release()

	type blockWithKey struct {
		Key string
		Blk block.Block
	}
	var blocks []blockWithKey

	for iter.Next() {
		key := iter.Key()
		if len(key) < 6 || string(key[:6]) != "block:" {
			continue // Only process actual block data
		}
		dec, err := storage.Decrypt(iter.Value())
		if err != nil {
			fmt.Printf("❌ Failed to decrypt value for key %s: %v\n", key, err)
			fmt.Printf("   Raw value (hex): %x\n", iter.Value())
			continue
		}
		var blk block.Block
		err = json.Unmarshal(dec, &blk)
		if err != nil {
			fmt.Printf("❌ Failed to decode block for key %s after decrypt: %v\n", key, err)
			fmt.Printf("   Decrypted value (string): %s\n", string(dec))
			fmt.Printf("   Decrypted value (hex): %x\n", dec)
			continue
		}
		blocks = append(blocks, blockWithKey{Key: string(key), Blk: blk})
	}

	// Sort by block height
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Blk.Height < blocks[j].Blk.Height
	})

	for _, b := range blocks {
		fmt.Printf("🗝️  Attempting to decode key: %s\n", b.Key)
		fmt.Printf("  ▸ Block Height: %d\n", b.Blk.Height)
		fmt.Printf("  ▸ BlockID: %x\n", b.Blk.BlockID[:])
		fmt.Printf("  ▸ PrevHash: %s\n", b.Blk.PrevHash)
		fmt.Println("-----------------------------------")
	}
}
