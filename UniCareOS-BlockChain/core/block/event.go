package block

import (
	"time"
	"fmt"
	"unicareos/types/ids"
)

type ChainedEvent struct {
	recordId string `json:"recordId,omitempty"` // Medical record unique identifier
	EventID         ids.ID
	EventType       string
	Description     string
	Timestamp       time.Time
	AuthorValidator ids.ID
	Memories        []MemorySubmission `json:"memories,omitempty"`
}

// ✅ Keep ONLY THIS here

type MemorySubmission struct {
	Content        string            `json:"content"`
	Tags           []string          `json:"tags,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Author         string            `json:"author"`
	Timestamp      time.Time         `json:"timestamp"`
	ParentID       string            `json:"parent_id,omitempty"`
	RevisionReason string            `json:"revision_reason,omitempty"`
}

// AttachMemoryToEvent attaches a validated memory to the event and logs the action for audit.
func (e *ChainedEvent) AttachMemoryToEvent(memory MemorySubmission) error {
	if err := ValidateMemoryPayload(memory); err != nil {
		return err
	}
	e.Memories = append(e.Memories, memory)
	// TODO: Add audit log entry here
	return nil
}

// ValidateMemoryPayload checks that the memory payload is valid and safe.
func ValidateMemoryPayload(memory MemorySubmission) error {
	if len(memory.Content) == 0 || len(memory.Content) > 4096 {
		return fmt.Errorf("memory content is required and must be less than 4096 characters")
	}
	if len(memory.Author) == 0 || len(memory.Author) > 128 {
		return fmt.Errorf("memory author is required and must be less than 128 characters")
	}
	if len(memory.ParentID) > 128 {
		return fmt.Errorf("parent_id field too long")
	}
	if len(memory.RevisionReason) > 256 {
		return fmt.Errorf("revision_reason field too long")
	}
	if memory.Tags != nil && len(memory.Tags) > 16 {
		return fmt.Errorf("too many tags (max 16)")
	}
	for _, tag := range memory.Tags {
		if len(tag) > 64 {
			return fmt.Errorf("tag too long (max 64 chars)")
		}
	}
	// Optionally: validate/sanitize metadata keys/values
	return nil
}





