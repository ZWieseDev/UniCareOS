package networking

import (
	"fmt"
	"encoding/hex"
	"unicareos/core/chain"

)

// HandleForkChoiceReorg performs fork-choice reorg using the chain/forkchoice logic
func (n *Network) HandleForkChoiceReorg(_ string, _ int, _ string) error {
	fmt.Printf("[REORG] Starting fork-choice reorg using all known peers from peer table\n")

	myHeight := n.getChainHeight()
	myTip := n.GetLatestBlockID()

	peers := n.Peers()
	var peerInfo []chain.PeerTipInfo
	for _, p := range peers {
		peerTipBytes, err := hex.DecodeString(p.TipBlockID)
		if err != nil || len(peerTipBytes) != 32 {
			fmt.Printf("[REORG] Skipping peer %s: invalid tip blockID %s\n", p.Address, p.TipBlockID)
			continue
		}
		var peerTipID [32]byte
		copy(peerTipID[:], peerTipBytes)
		peerInfo = append(peerInfo, chain.PeerTipInfo{
			Height:  p.ChainHeight,
			BlockID: peerTipID,
			Address: p.Address,
		})
	}

	fc := chain.NewForkChoice(n.store)
	return fc.CheckAndSync(myHeight, myTip, peerInfo)
}

// Optionally: You can call this from SaveNewBlock or orphan handling logic when a fork is detected.
