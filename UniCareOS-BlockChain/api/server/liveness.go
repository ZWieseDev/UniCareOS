// liveness.go - Liveness probe logic for UniCareOS Node
package server

// NodeLiveness returns true if the node is running and has produced at least one block.
func (s *Server) NodeLiveness() bool {
	metrics := s.GetNodeMetrics()
	return metrics.BlockHeight > 0
}
