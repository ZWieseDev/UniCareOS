// version.go - Node & API version info for UniCareOS Node
package server

// NodeVersion returns the current node software version.
func NodeVersion() string {
	// TODO: Return version from build flags or config
	return "v0.0.1-dev"
}

// APIVersion returns the current API version.
func APIVersion() string {
	// TODO: Return API version from config/constant
	return "v1"
}
