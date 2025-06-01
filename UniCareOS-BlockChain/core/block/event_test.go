package block

import (
	"testing"
	"time"
	"encoding/json"
)

func TestValidateMemoryPayload(t *testing.T) {
	valid := MemorySubmission{
		Content:   "Test memory content",
		Author:    "tester",
		Timestamp: time.Now(),
	}
	if err := ValidateMemoryPayload(valid); err != nil {
		t.Errorf("expected valid memory, got error: %v", err)
	}

	tooLong := MemorySubmission{Content: string(make([]byte, 4097)), Author: "tester", Timestamp: time.Now()}
	if err := ValidateMemoryPayload(tooLong); err == nil {
		t.Error("expected error for too long content, got nil")
	}

	noAuthor := MemorySubmission{Content: "ok", Author: "", Timestamp: time.Now()}
	if err := ValidateMemoryPayload(noAuthor); err == nil {
		t.Error("expected error for missing author, got nil")
	}
}

func TestAttachMemoryToEvent(t *testing.T) {
	evt := &ChainedEvent{
		EventID:   [32]byte{1,2,3},
		EventType: "memory",
		Timestamp: time.Now(),
	}
	mem := MemorySubmission{
		Content:   "Attach test",
		Author:    "tester",
		Timestamp: time.Now(),
	}
	err := evt.AttachMemoryToEvent(mem)
	if err != nil {
		t.Errorf("expected attach to succeed, got %v", err)
	}
	if len(evt.Memories) != 1 {
		t.Errorf("expected 1 memory attached, got %d", len(evt.Memories))
	}
	if evt.Memories[0].Content != mem.Content {
		t.Errorf("expected content '%s', got '%s'", mem.Content, evt.Memories[0].Content)
	}
}

func TestBlockSerialization(t *testing.T) {
	mem := MemorySubmission{
		Content:   "Serialize test",
		Author:    "tester",
		Timestamp: time.Now(),
	}
	evt := ChainedEvent{
		EventID:   [32]byte{4,5,6},
		EventType: "memory",
		Timestamp: time.Now(),
		Memories:  []MemorySubmission{mem},
	}
	block := Block{
		Version:   "1.0",
		Height:    42,
		Events:    []ChainedEvent{evt},
		Timestamp: time.Now(),
	}
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("failed to marshal block: %v", err)
	}
	var out Block
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Fatalf("failed to unmarshal block: %v", err)
	}
	if len(out.Events) != 1 || len(out.Events[0].Memories) != 1 {
		t.Errorf("expected 1 event with 1 memory, got %d events, %d memories", len(out.Events), len(out.Events[0].Memories))
	}
	if out.Events[0].Memories[0].Content != mem.Content {
		t.Errorf("expected memory content '%s', got '%s'", mem.Content, out.Events[0].Memories[0].Content)
	}
}
