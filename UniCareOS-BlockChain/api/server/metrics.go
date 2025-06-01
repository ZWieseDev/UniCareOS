// metrics.go - Metrics collection for UniCareOS Node
package server

import (
	"time"
	"runtime"
	"syscall"
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
)

// NodeMetrics holds granular health metrics for the node.
type NodeMetrics struct {
	UptimeSeconds   int64   `json:"uptime_seconds"`
	BlockHeight     int     `json:"block_height"`
	PeerCount       int     `json:"peer_count"`
	CPULoadPercent  float64 `json:"cpu_load_percent"`
	MemoryMB        float64 `json:"memory_mb"`
	DiskFreeMB      float64 `json:"disk_free_mb"`
	SyncLagSeconds  int64   `json:"sync_lag_seconds"`
	LastBlockTime   string  `json:"last_block_time"`
}

// Track server start time for uptime calculation
var startTime = time.Now()

// GetNodeMetrics returns current health metrics for the node.
func (s *Server) GetNodeMetrics() NodeMetrics {
	// DEBUG: Print pointers and values to verify wiring
	blockHeight := 0
	peerCount := 0
	if s.network != nil {
		blockHeight = s.network.GetChainHeight()
		peerCount = len(s.network.Peers())
	}
	fmt.Printf("\033[31m[DEBUG][metrics] blockHeight=%d, peerCount=%d, store=%p, network=%p\033[0m\n", blockHeight, peerCount, s.store, s.network)

	// Uptime
	uptime := int64(time.Since(startTime).Seconds())

	// Memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryMB := float64(m.Alloc) / (1024 * 1024)

	// Disk usage (root partition)
	var disk syscall.Statfs_t
	diskFreeMB := 0.0
	if err := syscall.Statfs("/", &disk); err == nil {
		diskFreeMB = float64(disk.Bfree) * float64(disk.Bsize) / (1024 * 1024)
	}

	// CPU usage: Use gopsutil to get current CPU percent
	cpuPercents, err := cpu.Percent(0, false)
	cpuLoad := 0.0
	if err == nil && len(cpuPercents) > 0 {
		cpuLoad = cpuPercents[0]
	}

	// Peer count, block height, last block time: These should be filled by the main server logic. Use stubs for now.
	// Block height: use s.network.GetChainHeight() if available, else fallback to storage
	blockHeight = 0
	if s.network != nil {
		blockHeight = s.network.GetChainHeight()
	}
	if blockHeight == 0 && s.store != nil {
		blockHeight, _ = s.store.GetChainHeight()
	}

	// Peer count: use s.network.Peers() for live peer count
	peerCount = 0
	if s.network != nil {
		peerList := s.network.Peers()
		peerCount = len(peerList)
	}

	// Last block time: get from storage if height > 0
	var lastBlockTime time.Time
	if blockHeight > 0 && s.store != nil {
		blk, err := s.store.GetBlockByHeight(blockHeight - 1)
		if err == nil {
			lastBlockTime = blk.Timestamp
		}
	}

	// Sync lag
	syncLag := int64(time.Since(lastBlockTime).Seconds())

	return NodeMetrics{
		UptimeSeconds: uptime,
		BlockHeight:   blockHeight,
		PeerCount:     peerCount,
		CPULoadPercent: cpuLoad,
		MemoryMB:      memoryMB,
		DiskFreeMB:    diskFreeMB,
		SyncLagSeconds: syncLag,
		LastBlockTime: lastBlockTime.Format(time.RFC3339),
	}
}
