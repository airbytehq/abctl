package auth

import (
	"time"
)

// needsRefresh checks if credentials need refreshing
func needsRefresh(creds *Credentials) bool {
	// Refresh if token expires within 5 minutes
	bufferTime := 5 * time.Minute
	return time.Now().After(creds.ExpiresAt.Add(-bufferTime))
}
