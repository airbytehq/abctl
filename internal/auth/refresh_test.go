package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNeedsRefresh(t *testing.T) {
	// Far future - no refresh needed
	creds := &Credentials{ExpiresAt: time.Now().Add(10 * time.Minute)}
	assert.False(t, needsRefresh(creds))

	// Near future (within 5min buffer) - refresh needed
	creds = &Credentials{ExpiresAt: time.Now().Add(3 * time.Minute)}
	assert.True(t, needsRefresh(creds))

	// Past - refresh needed
	creds = &Credentials{ExpiresAt: time.Now().Add(-1 * time.Minute)}
	assert.True(t, needsRefresh(creds))
}
