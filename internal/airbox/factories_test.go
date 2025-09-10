package airbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/airbytehq/abctl/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIService(t *testing.T) {
	tests := []struct {
		name          string
		setupConfig   func(t *testing.T, serverURL string) ConfigStore
		setupHandler  func(t *testing.T) http.HandlerFunc
		expectedError string
	}{
		{
			name: "success with valid config",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"access_token":"mock-token","token_type":"Bearer","expires_in":3600}`))
				}
			},
			setupConfig: func(t *testing.T, serverURL string) ConfigStore {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				_ = os.Setenv(EnvConfigPath, configPath)
				t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

				store := &FileConfigStore{}
				config := &Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: Context{
								AirbyteAPIURL:  serverURL,
								AirbyteURL:     "https://test.airbyte.com",
								OrganizationID: "org-123",
								Edition:        "cloud",
								Auth:           NewAuthWithOAuth2("client-id", "client-secret"),
							},
						},
					},
					Credentials: &auth.Credentials{
						AccessToken:  "test-token",
						RefreshToken: "refresh-token",
					},
				}

				err := store.Save(config)
				require.NoError(t, err)

				return store
			},
		},
		{
			name: "no credentials",
			setupConfig: func(t *testing.T, serverURL string) ConfigStore {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				_ = os.Setenv(EnvConfigPath, configPath)
				t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

				store := &FileConfigStore{}
				config := &Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: Context{
								AirbyteAPIURL: "https://api.test.airbyte.com",
								AirbyteURL:    "https://test.airbyte.com",
								Edition:       "cloud",
								Auth:          NewAuthWithOAuth2("client-id", "client-secret"),
							},
						},
					},
					Credentials: nil, // No credentials
				}

				err := store.Save(config)
				require.NoError(t, err)

				return store
			},
			expectedError: "no credentials - please run 'airbox auth login' first",
		},
		{
			name: "no current context",
			setupConfig: func(t *testing.T, serverURL string) ConfigStore {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				_ = os.Setenv(EnvConfigPath, configPath)
				t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

				store := &FileConfigStore{}
				config := &Config{
					CurrentContext: "", // No current context
					Contexts:       []NamedContext{},
					Credentials: &auth.Credentials{
						AccessToken: "test-token",
					},
				}

				err := store.Save(config)
				require.NoError(t, err)

				return store
			},
			expectedError: "no current context configured",
		},
		{
			name: "no auth configured",
			setupConfig: func(t *testing.T, serverURL string) ConfigStore {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				_ = os.Setenv(EnvConfigPath, configPath)
				t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

				store := &FileConfigStore{}
				config := &Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: Context{
								AirbyteAPIURL:  "https://api.test.airbyte.com",
								AirbyteURL:     "https://test.airbyte.com",
								OrganizationID: "org-123",
								Edition:        "cloud",
								Auth:           Auth{}, // No auth
							},
						},
					},
					Credentials: &auth.Credentials{
						AccessToken: "test-token",
					},
				}

				err := store.Save(config)
				require.NoError(t, err)

				return store
			},
			expectedError: "no auth configured - please run 'airbox config init' first",
		},
		{
			name: "success with empty organization",
			setupHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"access_token":"mock-token","token_type":"Bearer","expires_in":3600}`))
				}
			},
			setupConfig: func(t *testing.T, serverURL string) ConfigStore {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				_ = os.Setenv(EnvConfigPath, configPath)
				t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

				store := &FileConfigStore{}
				config := &Config{
					CurrentContext: "https://test.airbyte.com",
					Contexts: []NamedContext{
						{
							Name: "https://test.airbyte.com",
							Context: Context{
								AirbyteAPIURL:  serverURL,
								AirbyteURL:     "https://test.airbyte.com",
								OrganizationID: "", // No org
								Edition:        "cloud",
								Auth:           NewAuthWithOAuth2("client-id", "client-secret"),
							},
						},
					},
					Credentials: &auth.Credentials{
						AccessToken: "test-token",
					},
				}

				err := store.Save(config)
				require.NoError(t, err)

				return store
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set OAuth env vars for factory
			_ = os.Setenv("AIRBYTE_CLIENT_ID", "test-client-id")
			_ = os.Setenv("AIRBYTE_CLIENT_SECRET", "test-client-secret")
			t.Cleanup(func() {
				_ = os.Unsetenv("AIRBYTE_CLIENT_ID")
				_ = os.Unsetenv("AIRBYTE_CLIENT_SECRET")
			})

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

			configStore := tt.setupConfig(t, server.URL)

			httpClient := &http.Client{}

			service, err := NewAPIService(context.Background(), httpClient, configStore)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)

				// Verify it implements the interface
				// Service is already of type api.Service, no need to type assert
				assert.NotNil(t, service)
			}
		})
	}
}
