package networking

import (
    "time"
    "fmt"
)

// Rate limiting logic for the Network struct

const rateLimitWindow = 60 * time.Second
const maxRequestsPerWindow = 3000 // allows 100 requests per 1 minute (production)

// Progressive ban durations
var banDurations = []time.Duration{
	10 * time.Minute,
	1 * time.Hour,
	24 * time.Hour,
}
const permabanDuration = 100 * 365 * 24 * time.Hour // effectively permanent

// AllowPeerRequest checks and updates the rate limit for a peer
func (n *Network) AllowPeerRequest(address string) bool {
	if n.banCounts == nil {
		n.banCounts = make(map[string]int)
	}

	n.lock.Lock()
	defer n.lock.Unlock()
	if n.peerRequestCounts == nil {
		n.peerRequestCounts = make(map[string][]time.Time)
	}
	now := time.Now()
	times := n.peerRequestCounts[address]
	// Keep only recent requests
	var recent []time.Time
	for _, t := range times {
		if now.Sub(t) < rateLimitWindow {
			recent = append(recent, t)
		}
	}
	recent = append(recent, now)
	n.peerRequestCounts[address] = recent
	fmt.Printf("[DEBUG] %d requests from %s in last %s\n", len(recent), address, rateLimitWindow)
	if len(recent) > maxRequestsPerWindow {
		fmt.Printf("[RATE LIMIT] Would block request from %s (over limit)\n", address)
		// Only ban if not already banned
		if !n.IsPeerBanned(address) {
			// Progressive ban logic
			n.banCounts[address]++
			banCount := n.banCounts[address]
			if banCount > len(banDurations) {
				n.BanPeer(address, permabanDuration)
				fmt.Printf("[PERMABAN] Permanently banned %s after %d violations\n", address, banCount)
			} else {
				dur := banDurations[banCount-1]
				n.BanPeer(address, dur)
				fmt.Printf("[BAN] %s banned for %s (violation #%d)\n", address, dur, banCount)
			}
		}
		return false // Rate limited
	}
	return true
}
