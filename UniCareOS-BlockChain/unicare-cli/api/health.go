package api

import (
	"encoding/json"
	"net/http"
	"io"
	"fmt"
	"bytes"
)

type HealthMetrics struct {
	Status        string  `json:"status"`
	Metrics struct {
		UptimeSeconds   int64   `json:"uptime_seconds"`
		BlockHeight     int     `json:"block_height"`
		PeerCount       int     `json:"peer_count"`
		CPULoadPercent  float64 `json:"cpu_load_percent"`
		MemoryMB        float64 `json:"memory_mb"`
		DiskFreeMB      float64 `json:"disk_free_mb"`
		SyncLagSeconds  int64   `json:"sync_lag_seconds"`
		LastBlockTime   string  `json:"last_block_time"`
	} `json:"metrics"`
}

func GetHealthMetrics() (HealthMetrics, error) {
	resp, err := http.Get("http://localhost:8080/nodehealth")
	if err != nil {
		return HealthMetrics{}, err
	}
	defer resp.Body.Close()

	// DEBUG: Print the raw HTTP response body
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("RAW BODY:", string(body))
	resp.Body = io.NopCloser(bytes.NewBuffer(body)) // Reset for decoder

	var data HealthMetrics
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return HealthMetrics{}, err
	}
	return data, nil
}

func GetHealth() (string, error) {
	resp, err := http.Get("http://localhost:8080/nodehealth")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	b, _ := json.MarshalIndent(data, "", "  ")
	return string(b), nil
}

func GetLiveness() (bool, error) {
	resp, err := http.Get("http://localhost:8080/health/liveness")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var result struct { Alive bool `json:"alive"` }
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Alive, nil
}

func GetReadiness() (bool, error) {
	resp, err := http.Get("http://localhost:8080/health/readiness")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var result struct { Ready bool `json:"ready"` }
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Ready, nil
}
