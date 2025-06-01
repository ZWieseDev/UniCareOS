package chain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// FetchBlockFromPeerGET fetches a block as JSON from a peer using /get_block/{blockID}
func FetchBlockFromPeerGET(peerAddr string, blockID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s/get_block/%s", peerAddr, blockID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to GET block: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer returned status: %s", resp.Status)
	}
	var block map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&block)
	if err != nil {
		return nil, fmt.Errorf("failed to decode block JSON: %w", err)
	}
	return block, nil
}

// FetchBlockFromPeerPOST fetches a block as raw bytes using /request_block
func FetchBlockFromPeerPOST(peerAddr string, blockID string) ([]byte, error) {
	url := fmt.Sprintf("http://%s/request_block", peerAddr)
	payload := fmt.Sprintf(`{"blockID":"%s"}`, blockID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to POST block request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer returned status: %s", resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
}
