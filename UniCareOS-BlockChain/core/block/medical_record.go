package block

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
	"fmt"
	"unicareos/core/validation"
	"unicareos/types/ids"
	"encoding/json"
	"io/ioutil"
	"sync"
)

// MedicalRecordSubmission represents a medical record submission payload as per MCP 12.
type MedicalRecordSubmission struct {
	Record               map[string]interface{} `json:"record"` // The medical record fields (schema-compliant)
	Signature            string                 `json:"signature"` // Base64-encoded signature
	WalletAddress        string                 `json:"walletAddress"` // Provider wallet address
	RevisionOf           string                 `json:"revisionOf,omitempty"` // Optional prior record ID
	RevisionReason       string                 `json:"revisionReason,omitempty"` // Optional revision reason
	DocLineage           []string               `json:"docLineage,omitempty"` // Optional document lineage
	SubmissionTimestamp  time.Time              `json:"submissionTimestamp"` // RFC3339 timestamp
}

// TransactionReceipt represents the result of a medical record submission as per MCP 12.
type TransactionReceipt struct {
	TxID        string   `json:"txID"`           // Unique transaction ID
	BlockHash   string   `json:"blockHash"`      // Block hash containing the record
	BlockHeight uint64   `json:"blockHeight"`    // Height of the block
	Status      string   `json:"status"`         // "pending", "confirmed", or "failed"
	Errors      []string `json:"errors,omitempty"` // Error codes if any
}

// AuditLogEntry represents an audit log entry for a medical record submission as per MCP 12.
type AuditLogEntry struct {
	EventID     string    `json:"eventID"`      // Submission event ID
	SubmittedBy string    `json:"submittedBy"`  // Wallet address or identifier
	Timestamp   time.Time `json:"timestamp"`    // When the event occurred
	Status      string    `json:"status"`       // "accepted", "rejected", etc.
	Reason      string    `json:"reason"`       // Error or success reason/message
	PrevHash    string    `json:"prevHash"`     // Hash of previous entry (hex)
	EntryHash   string    `json:"entryHash"`    // Hash of this entry (hex)
}

// LogSubmissionTrace logs an audit entry for a submission event (in-memory and persistent)


func LogSubmissionTrace(block *Block, eventID, submittedBy, status, reason string, timestamp time.Time) {
	prevHash := ""
	if len(block.AuditLog) > 0 {
		prevHash = block.AuditLog[len(block.AuditLog)-1].EntryHash
	}
	tmpEntry := AuditLogEntry{
		EventID:     eventID,
		SubmittedBy: submittedBy,
		Timestamp:   timestamp,
		Status:      status,
		Reason:      reason,
		PrevHash:    prevHash,
	}
	// Calculate hash of this entry (excluding EntryHash itself)
	h := sha256.New()
	h.Write([]byte(tmpEntry.EventID))
	h.Write([]byte(tmpEntry.SubmittedBy))
	h.Write([]byte(tmpEntry.Timestamp.Format(time.RFC3339Nano)))
	h.Write([]byte(tmpEntry.Status))
	h.Write([]byte(tmpEntry.Reason))
	h.Write([]byte(tmpEntry.PrevHash))
	entryHash := hex.EncodeToString(h.Sum(nil))
	tmpEntry.EntryHash = entryHash

	block.AuditLog = append(block.AuditLog, tmpEntry)
	// Also log to persistent file for compliance
	validation.AuditValidationError(status, fmt.Sprintf("EventID=%s, Reason=%s, EntryHash=%s", eventID, reason, entryHash))
}

// SubmitRecordToBlock handles the submission of a medical record to a block, performing validation, duplicate/replay checks, block update, and logging.


var (
	authorizedWallets     = make(map[string]bool)
	authorizedWalletsLock sync.RWMutex
)

type walletEntry struct {
    Authorized bool   `json:"authorized"`
    PublicKey  string `json:"publicKey"`
}

func loadAuthorizedWallets() error {
    data, err := ioutil.ReadFile("core/block/authorized_wallets.json")
    if err != nil {
        return err
    }
    var wallets map[string]walletEntry
    if err := json.Unmarshal(data, &wallets); err != nil {
        return err
    }
    authorizedWalletsLock.Lock()
    defer authorizedWalletsLock.Unlock()
    authorizedWallets = make(map[string]bool)
    for k, v := range wallets {
        authorizedWallets[k] = v.Authorized
    }
    return nil
}

func ReloadAuthorizedWallets() error {
	return loadAuthorizedWallets()
}

func IsAuthorizedWallet(wallet string) bool {
	authorizedWalletsLock.RLock()
	defer authorizedWalletsLock.RUnlock()

	return authorizedWallets[wallet]
}

func init() {



	err := loadAuthorizedWallets()
	if err != nil {

	}

}




func SubmitRecordToBlock(submission MedicalRecordSubmission, block *Block) (TransactionReceipt, error) {
	if !IsAuthorizedWallet(submission.WalletAddress) {
		receipt := TransactionReceipt{
			TxID:        "",
			BlockHash:   "",
			BlockHeight: block.Height,
			Status:      "failed",
			Errors:      []string{"unauthorized_wallet: not in allowlist"},
		}
		LogSubmissionTrace(block, "", submission.WalletAddress, "unauthorized", "Unauthorized wallet address: "+submission.WalletAddress, submission.SubmissionTimestamp)
		return receipt, fmt.Errorf("unauthorized wallet: %s", submission.WalletAddress)
	}

	// 1. Validate the record (schema, required fields, etc.)
	if err := validation.ValidateRecord(submission.Record); err != nil {
		receipt := TransactionReceipt{
			TxID:        "", // Could generate a failed tx id if desired
			BlockHash:   "",
			BlockHeight: block.Height,
			Status:      "failed",
			Errors:      []string{"validation_failed: " + err.Error()},
		}
		LogSubmissionTrace(block, "", submission.WalletAddress, "failed", "Validation failed: "+err.Error(), submission.SubmissionTimestamp)
		return receipt, err
	}

	// 2. Verify the signature
	if err := validation.VerifyWalletSignature(submission.Record, submission.Signature, submission.WalletAddress); err != nil {
		receipt := TransactionReceipt{
			TxID:        "",
			BlockHash:   "",
			BlockHeight: block.Height,
			Status:      "failed",
			Errors:      []string{"invalid_signature: " + err.Error()},
		}
		LogSubmissionTrace(block, "", submission.WalletAddress, "invalid_signature", "Signature verification failed: "+err.Error(), submission.SubmissionTimestamp)
		return receipt, err
	}

	// Check if revision target exists (but do not block based on Finalized)
	if submission.RevisionOf != "" {
		foundOriginal := false
		for i := range block.Events {
			if block.Events[i].EventID.String() == submission.RevisionOf {
				foundOriginal = true
				break
			}
		}
		if !foundOriginal {
			errMsg := fmt.Sprintf("original record for revision not found: %s", submission.RevisionOf)
			LogSubmissionTrace(block, submission.RevisionOf, submission.WalletAddress, "rejected_revision_target_not_found", errMsg, submission.SubmissionTimestamp)
			receipt := TransactionReceipt{
				TxID:        "",
				BlockHash:   "",
				BlockHeight: block.Height,
				Status:      "failed",
				Errors:      []string{fmt.Sprintf("Original event %s for revision not found.", submission.RevisionOf)},
			}
			return receipt, fmt.Errorf(errMsg)
		}
	}

	// 3. Check for duplicate/replay
	if submission.Record != nil {
		recordId, _ := submission.Record["recordId"].(string)
		for _, evt := range block.Events {
			if evt.EventType == "medical_record" && evt.RecordID == recordId && recordId != "" {
				receipt := TransactionReceipt{
					TxID:        "",
					BlockHash:   "",
					BlockHeight: block.Height,
					Status:      "failed",
					Errors:      []string{"duplicate_submission: recordId already exists in block"},
				}
				LogSubmissionTrace(block, "", submission.WalletAddress, "duplicate", "Duplicate submission: recordId already exists in block", submission.SubmissionTimestamp)
				return receipt, fmt.Errorf("duplicate submission: recordId already exists in block")
			}
		}
	}

	// 4. Update the block with the new event/transaction
	recordId, _ := submission.Record["recordId"].(string)

	// --- Always build lineage recursively from revisionOf ---
	var docLineage []string
	if submission.RevisionOf != "" {
		visited := make(map[string]bool)
		currentID := submission.RevisionOf
		for len(currentID) > 0 && !visited[currentID] {
			visited[currentID] = true
			found := false
			for _, evt := range block.Events {
				if evt.EventID.String() == currentID {
					if evt.DocLineage != nil {
						docLineage = append(docLineage, evt.DocLineage...)
					}
					docLineage = append(docLineage, currentID)
					currentID = evt.RevisionOf
					found = true
					break
				}
			}
			if !found {
				break
			}
		}
	}
	// --- End lineage logic ---
	// Reverse to chronological order
	for i, j := 0, len(docLineage)-1; i < j; i, j = i+1, j-1 {
		docLineage[i], docLineage[j] = docLineage[j], docLineage[i]
	}
	event := ChainedEvent{
		RecordID: recordId,
		EventID:         generateEventID(submission), // Helper function to generate unique event ID
		EventType:       "medical_record",
		Description:     "Medical record submission",
		Timestamp:       submission.SubmissionTimestamp,
		AuthorValidator: ids.IDFromString(submission.WalletAddress), // Helper to convert address to ID
		RevisionReason:  submission.RevisionReason,
		RevisionOf:      submission.RevisionOf,
		DocLineage:      docLineage,
		Finalized:       false, // New events are not finalized by default
		// Add more fields as needed
	}
	block.Events = append(block.Events, event)
	// Debug log: print event struct as JSON
	
	if _, err := json.MarshalIndent(event, "", "  "); err == nil {
		//fmt.Printf("\033[33m[DEBUG ChainedEvent]\033[0m %s\n", string(debugJson))
	} else {
		//fmt.Printf("\033[31m[DEBUG ChainedEvent ERROR]\033[0m %v\n", err)
	}
	

	// --- Highlighted lineage log for revisions ---
	if submission.RevisionOf != "" {
		fmt.Printf("\033[33m[REVISION TRACKED]\033[0m eventID=%s revisionOf=%s reason=\"%s\" lineage=%v\n",
			event.EventID.String(), event.RevisionOf, event.RevisionReason, event.DocLineage)
	}
	// 5. Log the submission for audit/compliance
	LogSubmissionTrace(block, event.EventID.String(), submission.WalletAddress, "accepted", "Submission accepted and added to block", submission.SubmissionTimestamp)

	// 6. Return a TransactionReceipt (with status and errors if any)
	receipt := TransactionReceipt{
		TxID:        "", // TODO: generate unique transaction ID
		BlockHash:   "", // TODO: set after block inclusion
		BlockHeight: block.Height,
		Status:      "pending", // or "confirmed"/"failed" as appropriate
		Errors:      nil, // or error codes
	}
	return receipt, nil // TODO: handle errors and status
}

// generateEventID creates a unique event ID for a submission (stub).
func generateEventID(submission MedicalRecordSubmission) ids.ID {
	// TODO: Use a real hash of submission data for uniqueness
	return ids.NewID([]byte(submission.WalletAddress + submission.SubmissionTimestamp.String()))
}


// Optionally, add documentation or helper methods here as needed.
