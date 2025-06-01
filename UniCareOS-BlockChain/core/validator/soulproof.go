package validator

import (
	"unicareos/types/ids"
)

type SoulProof struct {
	ValidatorID ids.ID
	BlockID     ids.ID
	Signature   []byte
}
