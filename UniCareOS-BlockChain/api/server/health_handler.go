// health_handler.go - HTTP handler for /health, /health/liveness, /health/readiness
package server

import (
	"encoding/json"
	"net/http"
)

// HandleLiveness responds to /health/liveness
func (s *Server) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	resp := LivenessResponse{Alive: s.NodeLiveness()}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleReadiness responds to /health/readiness
func (s *Server) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	resp := ReadinessResponse{Ready: s.NodeReadiness()}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// NodeHealthResponse is the response type for the /nodehealth endpoint
 type NodeHealthResponse struct {
     Status  string      `json:"status"`
     Metrics NodeMetrics `json:"metrics"`
 }

// HandleNodeHealth responds to /nodehealth (summary health)
func (s *Server) HandleNodeHealth(w http.ResponseWriter, r *http.Request) {
    metrics := s.GetNodeMetrics()

    // Derive node health status from metrics (same as /status)
    status := "healthy"
    if metrics.BlockHeight == 0 {
        status = "initializing"
    } else if metrics.SyncLagSeconds > 30 {
        status = "syncing"
    } else if metrics.PeerCount == 0 {
        status = "isolated"
    }

    resp := NodeHealthResponse{
        Status:  status,
        Metrics: metrics,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
