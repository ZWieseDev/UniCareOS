package api

import (
    "encoding/json"
    "fmt"
    "net/http"
    "io/ioutil"
    "bytes"
)

type Status struct {
    Name   string `json:"name"`
    Status string `json:"status"`
    Height int    `json:"height"`
}

func (s Status) ToJSON() string {
    b, _ := json.MarshalIndent(s, "", "  ")
    return string(b)
}

func GetStatus() (Status, error) {
    resp, err := http.Get("http://localhost:8080/api/cli/status")
    if err != nil {
        return Status{}, err
    }
    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)
    var status Status
    if err := json.Unmarshal(body, &status); err != nil {
        return Status{}, err
    }
    return status, nil
}

// MemoryTx represents a transaction in the mempool.
type MemoryTx struct {
    Description string `json:"description"`
    Author      string `json:"author"`
    Emotion     string `json:"emotion"`
    ParentID    string `json:"parent_id"`
    // Add other fields as needed
}

// SubmitMemory submits a memory event to the node.
func SubmitMemory(description, author, emotion, parentID string) error {
    payload := map[string]string{
        "description": description,
        "author": author,
        "emotion": emotion,
        "parent_id": parentID,
    }
    b, _ := json.Marshal(payload)
    resp, err := http.Post("http://localhost:8080/submit_memory", "application/json", bytes.NewReader(b))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        body, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("Error: %s", string(body))
    }
    return nil
}

// GetMempool fetches the list of transactions in the mempool.
func GetMempool() ([]MemoryTx, error) {
    resp, err := http.Get("http://localhost:8080/api/cli/mempool")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)
    var txs []MemoryTx
    if err := json.Unmarshal(body, &txs); err != nil {
        return nil, err
    }
    return txs, nil
}