package genesis

import "time"

// ValidatorConfig represents a validator entry in the genesis config.
type ValidatorConfig struct {
	DID    string `json:"did"`
	PubKey string `json:"pubKey"`
	Bond   int    `json:"bond"`
}

// InitialParams holds chain parameters in the genesis config.
type InitialParams struct {
	TokenID           string `json:"tokenId"`
	ProtocolVersion   string `json:"protocolVersion"`
	BlockTime         int    `json:"blockTime,omitempty"`
	MaxBlockSize      int    `json:"maxBlockSize,omitempty"`
	ConfirmationDepth int    `json:"confirmationDepth,omitempty"`
}

// GenesisConfig represents the full genesis configuration schema.
type GenesisConfig struct {
	Signatures []string `json:"signatures"` // Simulated multi-party signatures
	ChainID          string            `json:"chainId"`
	GenesisTime      time.Time         `json:"genesisTime"`
	InitialValidators []ValidatorConfig `json:"initialValidators"`
	InitialParams    InitialParams     `json:"initialParams"`
	InitialSchemaHash string           `json:"initialSchemaHash,omitempty"`
}
