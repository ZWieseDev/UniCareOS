package validation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"os"
	"path/filepath"
	"time"
	"encoding/base64"
	"unicode/utf8"
	"github.com/xeipuuv/gojsonschema"
)


func getSchemaPath(schemaVersion string) string {
	if env := os.Getenv("MEDICAL_SCHEMA_PATH"); env != "" {
		return env
	}
	switch schemaVersion {
	case "1.0", "1":
		return filepath.Join("core", "validation", "schemas", "medical_record_schema_v1.json")
	case "2.0", "2":
		return filepath.Join("core", "validation", "schemas", "medical_record_schema_v2.json")
	default:
		return filepath.Join("core", "validation", "schemas", "medical_record_schema_v1.json")
	}
}



// ValidateMedicalPayload validates a raw JSON payload against the schema and additional logic
func ValidateMedicalPayload(payload []byte) error {
	// Unmarshal for schemaVersion
	var rec map[string]interface{}
	if err := json.Unmarshal(payload, &rec); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	schemaVersion, _ := rec["schemaVersion"].(string)
	schemaPath := getSchemaPath(schemaVersion)
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)

	// Validate against JSON Schema
	documentLoader := gojsonschema.NewBytesLoader(payload)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}
	if !result.Valid() {
		// Aggregate schema validation errors
		errStr := ""
		for _, e := range result.Errors() {
			errStr += e.String() + "; "
		}
		return fmt.Errorf("payload failed schema validation: %s", errStr)
	}

	// Security/privacy checks: base64 validation and maxLength
	base64Fields := []string{"patientId", "providerID", "payloadSignature", "notes"}
	for _, field := range base64Fields {
		if val, ok := rec[field].(string); ok && val != "" {
			if _, err := base64.StdEncoding.DecodeString(val); err != nil {
				AuditValidationError("base64_check", fmt.Sprintf("%s is not valid base64", field))
				return fmt.Errorf("%s is not valid base64", field)
			}
		}
	}

	// Explicit regex check for patientDID (if not already enforced by schema)
	if did, ok := rec["patientDID"].(string); ok && did != "" {
		// Regex: ^did:[a-z0-9]+:[a-zA-Z0-9.-]+$
		matched := false
		if len(did) > 0 {
			re := `^did:[a-z0-9]+:[a-zA-Z0-9.-]+$`
			matched, _ = regexp.MatchString(re, did)
		}
		if !matched {
			AuditValidationError("regex_check", "patientDID does not match DID pattern")
			return fmt.Errorf("patientDID does not match DID pattern")
		}
	}

	// encryptionContext.iv and tag
	if ctx, ok := rec["encryptionContext"].(map[string]interface{}); ok {
		for _, sub := range []string{"iv", "tag"} {
			if sval, ok := ctx[sub].(string); ok && sval != "" {
				if _, err := base64.StdEncoding.DecodeString(sval); err != nil {
					AuditValidationError("base64_check", fmt.Sprintf("encryptionContext.%s is not valid base64", sub))
					return fmt.Errorf("encryptionContext.%s is not valid base64", sub)
				}
			}
		}
	}
	// maxLength checks
	if docHash, ok := rec["docHash"].(string); ok && utf8.RuneCountInString(docHash) > 64 {
		AuditValidationError("length_check", "docHash exceeds 64 characters")
		return fmt.Errorf("docHash exceeds 64 characters")
	}
	if ps, ok := rec["payloadSignature"].(string); ok && utf8.RuneCountInString(ps) > 512 {
		AuditValidationError("length_check", "payloadSignature exceeds 512 characters")
		return fmt.Errorf("payloadSignature exceeds 512 characters")
	}
	if notes, ok := rec["notes"].(string); ok && utf8.RuneCountInString(notes) > 1024 {
		AuditValidationError("length_check", "notes exceeds 1024 characters")
		return fmt.Errorf("notes exceeds 1024 characters")
	}

	// Check issuedAt format
	issuedAt, _ := rec["issuedAt"].(string)
	if err := EnforceTimestampFormat(issuedAt); err != nil {
		return err
	}

	return nil
}

// IsValidRecordType checks if the recordType is allowed
func IsValidRecordType(recordType string) bool {
	switch recordType {
	case "lab_result", "imaging", "discharge_summary":
		return true
	default:
		return false
	}
}

// CheckRequiredFields can be extended for conditional logic
func CheckRequiredFields(rec map[string]interface{}) error {
	required := []string{"recordId", "patientDID", "providerID", "schemaVersion", "recordType", "docHash", "issuedAt", "signedBy", "consentStatus", "dataProvenance", "retentionPolicy", "encryptionContext", "payloadSignature"}
	for _, k := range required {
		if _, ok := rec[k]; !ok {
			return fmt.Errorf("missing required field: %s", k)
		}
	}
	return nil
}

// EnforceTimestampFormat checks if issuedAt is RFC3339
func EnforceTimestampFormat(issuedAt string) error {
	if issuedAt == "" {
		return fmt.Errorf("issuedAt is empty")
	}
	if _, err := time.Parse(time.RFC3339, issuedAt); err != nil {
		return fmt.Errorf("issuedAt must be RFC3339: %w", err)
	}
	return nil
}

// Optionally add more custom validation functions below, e.g.,
// - Validate consentStatus transitions
// - Check encrypted fields are base64-encoded
// - Cross-verify signedBy against whitelist
