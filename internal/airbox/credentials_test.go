package airbox

import (
	"testing"

	"github.com/airbytehq/abctl/internal/auth"
	"github.com/stretchr/testify/assert"
)

func TestNewCredentialStoreAdapter(t *testing.T) {
	configStore := &FileConfigStore{}
	adapter := NewCredentialStoreAdapter(configStore)

	assert.NotNil(t, adapter)
}

func TestCredentialStoreAdapter_Load(t *testing.T) {
	tests := []struct {
		name          string
		setupConfig   func() *Config
		expectedCreds *auth.Credentials
		expectedError string
	}{
		{
			name: "valid credentials",
			setupConfig: func() *Config {
				return &Config{
					Credentials: &auth.Credentials{
						AccessToken:  "test-token",
						RefreshToken: "refresh-token",
					},
				}
			},
			expectedCreds: &auth.Credentials{
				AccessToken:  "test-token",
				RefreshToken: "refresh-token",
			},
		},
		{
			name: "no credentials",
			setupConfig: func() *Config {
				return &Config{
					Credentials: nil,
				}
			},
			expectedError: "no user credentials found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock config store
			mockStore := &mockConfigStore{
				config: tt.setupConfig(),
			}

			adapter := NewCredentialStoreAdapter(mockStore)
			creds, err := adapter.Load()

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, creds)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCreds, creds)
			}
		})
	}
}

func TestCredentialStoreAdapter_Save(t *testing.T) {
	mockStore := &mockConfigStore{
		config: &Config{
			Credentials: nil,
		},
	}

	adapter := NewCredentialStoreAdapter(mockStore)
	newCreds := &auth.Credentials{
		AccessToken:  "new-token",
		RefreshToken: "new-refresh",
	}

	err := adapter.Save(newCreds)
	assert.NoError(t, err)

	// Verify credentials were set in config
	assert.Equal(t, newCreds, mockStore.config.Credentials)
}

// mockConfigStore for testing credentials
type mockConfigStore struct {
	config *Config
	err    error
}

func (m *mockConfigStore) Load() (*Config, error) {
	return m.config, m.err
}

func (m *mockConfigStore) Save(*Config) error {
	return nil
}

func (m *mockConfigStore) GetPath() string {
	return "/tmp/test-config.yaml"
}

func (m *mockConfigStore) Exists() bool {
	return m.config != nil
}
