// status_handler.go - HTTP handler for /status
package server

import (
	"encoding/json"
	"net/http"
)

// HandleStatus responds to /status with node status
func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	metrics := s.GetNodeMetrics()

	// Derive node health status from metrics
	status := "healthy"
	if metrics.BlockHeight == 0 {
		status = "initializing"
	} else if metrics.SyncLagSeconds > 30 {
		status = "syncing"
	} else if metrics.PeerCount == 0 {
		status = "isolated"
	}

	resp := StatusResponse{
		Status:      status,
		Uptime:      metrics.UptimeSeconds,
		BlockHeight: metrics.BlockHeight,
		PeerCount:   metrics.PeerCount,
		Version:     NodeVersion(),
		APIVersion:  APIVersion(),
		LastBlock:   metrics.LastBlockTime,
		Metrics:     metrics,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
