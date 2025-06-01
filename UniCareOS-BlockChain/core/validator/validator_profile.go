package validator

import (
	"unicareos/types/ids"
)

type ValidatorProfile struct {
	ValidatorID     ids.ID
	PublicKey       []byte
	SoulWeight      int
	ParticipationRecord []ids.ID // List of Block IDs they helped validate
}
