package block

import "time"

// FinalizedEvent represents the off-chain, encrypted finalized event payload structure
// All PHI must be encrypted before being stored or referenced.
type FinalizedEvent struct {
    EventID        string    `json:"eventID"`
    PatientID      string    `json:"patientId"`      // Encrypted!
    SignedBy       string    `json:"signedBy"`
    RecordType     string    `json:"recordType"`
    DocHash        string    `json:"docHash"`        // Hash of encrypted doc
    PayloadHash    string    `json:"payloadHash"`    // Hash of encrypted payload
    Timestamp      time.Time `json:"timestamp"`
    RevisionOf     string    `json:"revisionOf,omitempty"`
    RevisionReason string    `json:"revisionReason,omitempty"`
    DocLineage     []string  `json:"docLineage,omitempty"`
    Tags           []string  `json:"tags,omitempty"`
    Notes          string    `json:"notes,omitempty"` // Encrypted!
}
