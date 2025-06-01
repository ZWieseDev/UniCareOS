package networking

import (
	"time"
	"fmt"
	"encoding/binary"
)

// Peer banning logic for the Network struct

// BanPeer bans a peer for a given duration
// NOTE: Assumes caller holds n.lock!
func (n *Network) BanPeer(address string, duration time.Duration) {
	if n.bannedPeers == nil {
		n.bannedPeers = make(map[string]time.Time)
	}
	expiry := time.Now().Add(duration)
	n.bannedPeers[address] = expiry
	fmt.Printf("[BAN] Peer %s banned for %s (until %s, now %s)\n", address, duration, expiry.Format(time.RFC3339), time.Now().Format(time.RFC3339))
	// --- Persistent Ban State ---
	if n.store != nil && n.store.DB() != nil {
		err := n.store.DB().Put([]byte("ban:"+address), []byte(expiry.Format(time.RFC3339)), nil)
		if err != nil {
			fmt.Printf("[ERROR] Failed to persist ban for %s: %v\n", address, err)
		}
		count := n.banCounts[address]
		countBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(countBytes, uint64(count))
		err = n.store.DB().Put([]byte("banCount:"+address), countBytes, nil)
		if err != nil {
			fmt.Printf("[ERROR] Failed to persist ban count for %s: %v\n", address, err)
		}
	}
}

// IsPeerBanned checks if a peer is currently banned
// NOTE: Assumes caller holds n.lock!
func (n *Network) IsPeerBanned(address string) bool {
	if n.bannedPeers == nil {
		return false
	}
	expiry, ok := n.bannedPeers[address]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		fmt.Printf("[UNBAN] Ban expired for %s (was until %s, now %s)\n", address, expiry.Format(time.RFC3339), time.Now().Format(time.RFC3339))
		fmt.Printf("[UNBAN DEBUG] Exact address: %s, expiry: %s, now: %s\n", address, expiry.Format(time.RFC3339), time.Now().Format(time.RFC3339))
		delete(n.bannedPeers, address)
		// --- Remove persistent ban state ---
		if n.store != nil && n.store.DB() != nil {
			err := n.store.DB().Delete([]byte("ban:"+address), nil)
			if err != nil {
				fmt.Printf("[ERROR] Failed to remove persistent ban for %s: %v\n", address, err)
			}
			err = n.store.DB().Delete([]byte("banCount:"+address), nil)
			if err != nil {
				fmt.Printf("[ERROR] Failed to remove persistent ban count for %s: %v\n", address, err)
			}
		}
		return false
	} else {
		fmt.Printf("[BAN CHECK] Peer %s is still banned (until %s, now %s)\n", address, expiry.Format(time.RFC3339), time.Now().Format(time.RFC3339))
		fmt.Printf("[BAN CHECK DEBUG] Exact address: %s, expiry: %s, now: %s\n", address, expiry.Format(time.RFC3339), time.Now().Format(time.RFC3339))
	}
	return true
}
