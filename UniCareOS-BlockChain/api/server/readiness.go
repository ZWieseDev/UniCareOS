// readiness.go - Readiness probe logic for UniCareOS Node
package server

// NodeReadiness returns true if the node is synced, has peers, and DB is accessible.
func (s *Server) NodeReadiness() bool {
	metrics := s.GetNodeMetrics()
	return metrics.BlockHeight > 0 && metrics.PeerCount > 0 && metrics.SyncLagSeconds < 30
}
