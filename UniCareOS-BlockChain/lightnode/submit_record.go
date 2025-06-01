package main

import (
	"os"
	"net/http"
	"io/ioutil"
	"bytes"
	"fmt"
)

func main() {
	// Read Ethos token from environment
	ethosToken := os.Getenv("ETHOS_TOKEN")
	if ethosToken == "" {
		fmt.Println("ETHOS_TOKEN not set in environment")
		os.Exit(1)
	}

	// Read API JWT secret from environment (optional, for Authorization header)
	apiJwtSecret := os.Getenv("API_JWT_SECRET")
	if apiJwtSecret == "" {
		fmt.Println("API_JWT_SECRET not set in environment")
		os.Exit(1)
	}

	// Prepare request
	url := "http://localhost:8080/api/v1/submit-medical-record"
	jsonData, err := ioutil.ReadFile("signed_record.json")
	if err != nil {
		fmt.Printf("Failed to read signed_record.json: %v\n", err)
		os.Exit(1)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiJwtSecret)
	req.Header.Set("X-Ethos-Token", ethosToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Response:", string(body))
}
