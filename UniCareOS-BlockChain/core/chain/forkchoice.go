package chain

import (
	"fmt"
	"strings"
	"encoding/hex"
	"unicareos/core/block"
	"unicareos/core/storage"
)

// ForkChoice handles chain sync and fork switching
// Call CheckAndSync on a schedule or after receiving new peer info.
type ForkChoice struct {
	Store *storage.Storage
}

// NewForkChoice returns a new ForkChoice instance
func NewForkChoice(store *storage.Storage) *ForkChoice {
	return &ForkChoice{Store: store}
}

// PeerTipInfo represents a peer's chain tip
// (Extend as needed to include peer ID, etc)
type PeerTipInfo struct {
	Height int
	BlockID [32]byte
	Address string // Peer address for block fetching
}

// CheckAndSync checks if any peer has a longer chain and, if so, reorgs to it using fork-choice logic
func (fc *ForkChoice) CheckAndSync(myHeight int, myTip [32]byte, peers []PeerTipInfo) error {
	longest := myHeight
	var bestPeer *PeerTipInfo
	for i, p := range peers {
		if p.Height > longest {
			longest = p.Height
			bestPeer = &peers[i]
		}
	}
	if bestPeer == nil {
		fmt.Println("[FORKCHOICE] No longer chain found; staying on current tip.")
		return nil
	}
	fmt.Printf("[FORKCHOICE] Found longer chain at peer (height %d). Performing fork-choice reorg...\n", bestPeer.Height)

	// 1. Build ancestor set for our chain
	myAncestors := map[[32]byte]int{} // block ID -> height
	current := myTip
	fmt.Println("[FORKCHOICE-DEBUG] Walking back our chain:")
	for h := myHeight; h >= 0; h-- {
		fmt.Printf("  Height %d: %x\n", h, current[:])
		myAncestors[current] = h
		blkBytes, err := fc.Store.GetBlock(current[:])
		if err != nil { break }
		blk, err := block.Deserialize(blkBytes)
		if err != nil { break }
		if blk.PrevHash == "" || blk.PrevHash == strings.Repeat("0", len(blk.PrevHash)) { break }
		prev, err := hex.DecodeString(blk.PrevHash)
		if err != nil || len(prev) != 32 { break }
		copy(current[:], prev)
	}

	// 2. Walk back peer's chain to find fork point and collect blocks to apply
	peerBlocks := make([][32]byte, 0, bestPeer.Height-myHeight)
	peerTip := bestPeer.BlockID
	forkPoint := [32]byte{}
	curHeight := bestPeer.Height
	for {
		if _, ok := myAncestors[peerTip]; ok {
			forkPoint = peerTip
			break
		}
		peerBlocks = append([][32]byte{peerTip}, peerBlocks...)
		blkBytes, err := FetchBlockFromPeerPOST(bestPeer.Address, fmt.Sprintf("%x", peerTip[:]))
		if err != nil {
			return fmt.Errorf("[FORKCHOICE] Failed to fetch block from peer: %v", err)
		}
		blk, err := block.Deserialize(blkBytes)
		if err != nil {
			return fmt.Errorf("[FORKCHOICE] Failed to deserialize peer block: %v", err)
		}
		if blk.PrevHash == "" || blk.PrevHash == strings.Repeat("0", len(blk.PrevHash)) { break }
		prev, err := hex.DecodeString(blk.PrevHash)
		if err != nil || len(prev) != 32 { break }
		copy(peerTip[:], prev)
		curHeight--
		if curHeight < 0 { break }
	}
	if isZero(forkPoint) {
		return fmt.Errorf("[FORKCHOICE] No common ancestor found; cannot reorg")
	}
	fmt.Printf("[FORKCHOICE] Fork point found at %x\n", forkPoint[:])

	// 3. Roll back to fork point
	err := fc.Store.RollbackToBlock(forkPoint)
	if err != nil {
		return fmt.Errorf("[FORKCHOICE] Rollback failed: %v", err)
	}
	fmt.Printf("[FORKCHOICE] Rolled back to fork point %x\n", forkPoint[:])

	// 4. Apply new blocks from fork point to peer tip
	for _, id := range peerBlocks {
		blkBytes, err := FetchBlockFromPeerPOST(bestPeer.Address, fmt.Sprintf("%x", id[:]))
		if err != nil {
			return fmt.Errorf("[FORKCHOICE] Failed to fetch block from peer: %v", err)
		}
		err = fc.Store.SaveBlock(id[:], blkBytes)
		if err != nil {
			return fmt.Errorf("[FORKCHOICE] Failed to save block: %v", err)
		}
		fmt.Printf("[FORKCHOICE] Applied block %x\n", id[:])
	}
	fmt.Println("[FORKCHOICE] Reorg complete. Now at height", longest)
	return nil
}

func isZero(id [32]byte) bool {
	for _, b := range id { if b != 0 { return false } }
	return true
}


// (You will need to implement fetchBlockFromPeer and add real peer/network logic.)
