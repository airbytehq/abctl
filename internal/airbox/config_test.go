package airbox

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_IsAuthenticated(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		expectedError string
	}{
		{
			name: "authenticated with valid token",
			config: &Config{
				Credentials: &Credentials{
					AccessToken: "valid-token",
				},
			},
		},
		{
			name:          "not authenticated - nil credentials",
			config:        &Config{Credentials: nil},
			expectedError: "not authenticated - please run 'airbox auth login' first",
		},
		{
			name: "not authenticated - empty access token",
			config: &Config{
				Credentials: &Credentials{
					AccessToken: "",
				},
			},
			expectedError: "not authenticated - please run 'airbox auth login' first",
		},
		{
			name: "authenticated with expired token",
			config: &Config{
				Credentials: &Credentials{
					AccessToken:  "valid-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Now().Add(-time.Hour), // Expired
				},
			},
			// Note: IsAuthenticated only checks for presence of token, not expiration
			// Token expiration is handled by the auth client
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.IsAuthenticated()

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}