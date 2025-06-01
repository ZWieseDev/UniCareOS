package block

import (
	"encoding/base64"
)

// EncodeToBase64 encodes a byte slice to a base64 string
func EncodeToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes a base64 string to a byte slice
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// IsValidUUID checks if a string is a valid UUID v4
func IsValidUUID(uuid string) bool {
	if len(uuid) != 36 {
		return false
	}
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		return false
	}
	// Check version (4) and variant (2) bits
	if uuid[14] != '4' || (uuid[19] != '8' && uuid[19] != '9' && uuid[19] != 'a' && uuid[19] != 'b') {
		return false
	}
	// Check remaining characters are valid hex
	for i, c := range uuid {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}
