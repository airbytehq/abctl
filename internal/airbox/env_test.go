package airbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/airbytehq/abctl/internal/auth"
	internalhttp "github.com/airbytehq/abctl/internal/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadOAuthEnvConfig(t *testing.T) {
	tests := []struct {
		name           string
		setupEnv       func()
		expectedID     string
		expectedSecret string
		expectedError  string
	}{
		{
			name: "valid env vars",
			setupEnv: func() {
				_ = os.Setenv("AIRBYTE_CLIENT_ID", "test-client-id")
				_ = os.Setenv("AIRBYTE_CLIENT_SECRET", "test-client-secret")
			},
			expectedID:     "test-client-id",
			expectedSecret: "test-client-secret",
		},
		{
			name: "missing client ID",
			setupEnv: func() {
				_ = os.Unsetenv("AIRBYTE_CLIENT_ID")
				_ = os.Setenv("AIRBYTE_CLIENT_SECRET", "test-client-secret")
			},
			expectedError: "required key AIRBYTE_CLIENT_ID missing value",
		},
		{
			name: "missing client secret",
			setupEnv: func() {
				_ = os.Setenv("AIRBYTE_CLIENT_ID", "test-client-id")
				_ = os.Unsetenv("AIRBYTE_CLIENT_SECRET")
			},
			expectedError: "required key AIRBYTE_CLIENT_SECRET missing value",
		},
		{
			name: "both missing",
			setupEnv: func() {
				_ = os.Unsetenv("AIRBYTE_CLIENT_ID")
				_ = os.Unsetenv("AIRBYTE_CLIENT_SECRET")
			},
			expectedError: "required key AIRBYTE_CLIENT_ID missing value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up env vars before and after test
			defer func() {
				_ = os.Unsetenv("AIRBYTE_CLIENT_ID")
				_ = os.Unsetenv("AIRBYTE_CLIENT_SECRET")
			}()

			tt.setupEnv()

			cfg, err := LoadOAuthEnvConfig()

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, cfg.ClientID)
				assert.Equal(t, tt.expectedSecret, cfg.ClientSecret)
			}
		})
	}
}

func TestOAuthEnvConfig_ToAuthClient(t *testing.T) {
	tests := []struct {
		name          string
		config        OAuthEnvConfig
		setupConfig   func(t *testing.T) ConfigStore
		setupHandler  func(t *testing.T) http.HandlerFunc
		expectedError string
	}{
		{
			name: "success",
			config: OAuthEnvConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
				}
			},
			setupConfig: func(t *testing.T) ConfigStore {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				_ = os.Setenv(EnvConfigPath, configPath)
				t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

				store := &FileConfigStore{}
				config := &Config{
					Credentials: &auth.Credentials{
						AccessToken: "existing-token",
					},
				}
				_ = store.Save(config)
				return store
			},
		},
		{
			name: "auth provider creation fails",
			config: OAuthEnvConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
				}
			},
			setupConfig: func(t *testing.T) ConfigStore {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				_ = os.Setenv(EnvConfigPath, configPath)
				t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

				store := &FileConfigStore{}
				config := &Config{}
				_ = store.Save(config)
				return store
			},
			expectedError: "invalid_client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup handler - default to failure if HTTP calls shouldn't happen
			handler := tt.setupHandler
			if handler == nil {
				handler = func(t *testing.T) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						t.Fatal("HTTP call should not be made for this test case")
					}
				}
			}

			// Create test server
			server := httptest.NewServer(handler(t))
			t.Cleanup(server.Close)

			configStore := tt.setupConfig(t)
			httpClient, err := internalhttp.NewClient(server.URL, &http.Client{})
			require.NoError(t, err)

			provider, err := tt.config.ToAuthClient(context.Background(), httpClient, configStore)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}
