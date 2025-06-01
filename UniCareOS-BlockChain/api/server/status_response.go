// status_response.go - JSON response structs for status/health endpoints
package server



// StatusResponse represents the JSON structure for /status endpoint
 type StatusResponse struct {
	Status      string      `json:"status"`
	Uptime      int64       `json:"uptime_seconds"`
	BlockHeight int         `json:"block_height"`
	PeerCount   int         `json:"peer_count"`
	Version     string      `json:"version"`
	APIVersion  string      `json:"api_version"`
	LastBlock   string      `json:"last_block_time"`
	Metrics     NodeMetrics `json:"metrics"`
}

// LivenessResponse for /health/liveness
 type LivenessResponse struct {
	Alive bool `json:"alive"`
}

// ReadinessResponse for /health/readiness
 type ReadinessResponse struct {
	Ready bool `json:"ready"`
}
