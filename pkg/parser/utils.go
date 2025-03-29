package parser

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

// generateTransactionID creates a simple unique ID based on date and payee
func generateTransactionID(date time.Time, payee string) string {
	// Clean up payee - remove spaces and convert to lowercase
	cleanPayee := strings.ToLower(strings.TrimSpace(payee))

	// Create a string combining date and payee
	input := fmt.Sprintf("%s-%s", date.Format("2006-01-02"), cleanPayee)

	// Generate SHA256 hash and take first 8 characters
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)[:8]
}
