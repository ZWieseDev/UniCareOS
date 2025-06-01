package validation

import (
	"encoding/json"
	"fmt"
	"os"
)

// ValidateRecord validates a medical record map using the existing payload validator.
func ValidateRecord(record map[string]interface{}) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("could not marshal record to JSON: %w", err)
	}

	// DEBUG: Print schema path and FULL contents
	schemaPath := os.Getenv("MEDICAL_SCHEMA_PATH")
	// fmt.Println("[DEBUG] MEDICAL_SCHEMA_PATH:", schemaPath) // Cleaned up for production
	if schemaPath != "" {
		_, err := os.ReadFile(schemaPath)
		if err != nil {
			// fmt.Println("[DEBUG] Error reading schema:", err) // Cleaned up for production
		} else {
			// fmt.Println("[DEBUG] FULL schema contents:", string(schemaBytes)) // Cleaned up for production
		}
	}

	// DEBUG: Print the full payload being validated
	// fmt.Println("[DEBUG] Payload being validated:", string(payload)) // Cleaned up for production

	return ValidateMedicalPayload(payload)
}
