package airbox

import (
	"testing"

	"github.com/airbytehq/abctl/internal/auth"
	"github.com/stretchr/testify/assert"
)

func TestContext_Validate(t *testing.T) {
	tests := []struct {
		name          string
		context       Context
		expectedError string
	}{
		{
			name: "valid context",
			context: Context{
				AirbyteAPIURL: "https://api.airbyte.com",
				AirbyteURL:    "https://cloud.airbyte.com",
				Edition:       "cloud",
				Auth:          NewAuthWithOAuth2("client-id", "client-secret"),
			},
		},
		{
			name: "missing AirbyteAPIURL",
			context: Context{
				AirbyteURL: "https://cloud.airbyte.com",
				Edition:    "cloud",
				Auth:       NewAuthWithOAuth2("client-id", "client-secret"),
			},
			expectedError: "airbyteApiUrl is required",
		},
		{
			name: "invalid AirbyteAPIURL",
			context: Context{
				AirbyteAPIURL: "://invalid-scheme",
				AirbyteURL:    "https://cloud.airbyte.com",
				Edition:       "cloud",
				Auth:          NewAuthWithOAuth2("client-id", "client-secret"),
			},
			expectedError: "invalid airbyteApiUrl",
		},
		{
			name: "missing AirbyteURL",
			context: Context{
				AirbyteAPIURL: "https://api.airbyte.com",
				Edition:       "cloud",
				Auth:          NewAuthWithOAuth2("client-id", "client-secret"),
			},
			expectedError: "airbyteUrl is required",
		},
		{
			name: "invalid AirbyteURL",
			context: Context{
				AirbyteAPIURL: "https://api.airbyte.com",
				AirbyteURL:    "://invalid",
				Edition:       "cloud",
				Auth:          NewAuthWithOAuth2("client-id", "client-secret"),
			},
			expectedError: "invalid airbyteUrl",
		},
		{
			name: "missing edition",
			context: Context{
				AirbyteAPIURL: "https://api.airbyte.com",
				AirbyteURL:    "https://cloud.airbyte.com",
				Auth:          NewAuthWithOAuth2("client-id", "client-secret"),
			},
			expectedError: "edition is required",
		},
		{
			name: "invalid auth - missing client ID",
			context: Context{
				AirbyteAPIURL: "https://api.airbyte.com",
				AirbyteURL:    "https://cloud.airbyte.com",
				Edition:       "cloud",
				Auth:          NewAuthWithOAuth2("", "client-secret"),
			},
			expectedError: "OAuth2 auth missing required 'clientId' field",
		},
		{
			name: "invalid auth - missing client secret",
			context: Context{
				AirbyteAPIURL: "https://api.airbyte.com",
				AirbyteURL:    "https://cloud.airbyte.com",
				Edition:       "cloud",
				Auth:          NewAuthWithOAuth2("client-id", ""),
			},
			expectedError: "OAuth2 auth missing required 'clientSecret' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.context.Validate()
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_GetCurrentContext(t *testing.T) {
	validContext := Context{
		AirbyteAPIURL: "https://api.airbyte.com",
		AirbyteURL:    "https://cloud.airbyte.com",
		Edition:       "cloud",
		Auth:          NewAuthWithOAuth2("client-id", "client-secret"),
	}

	tests := []struct {
		name          string
		config        Config
		expectedCtx   *Context
		expectedError string
	}{
		{
			name: "valid current context",
			config: Config{
				CurrentContext: "https://cloud.airbyte.com",
				Contexts: []NamedContext{
					{Name: "https://cloud.airbyte.com", Context: validContext},
				},
			},
			expectedCtx: &validContext,
		},
		{
			name: "no current context",
			config: Config{
				Contexts: []NamedContext{
					{Name: "https://cloud.airbyte.com", Context: validContext},
				},
			},
			expectedError: "no current context set",
		},
		{
			name: "current context not found",
			config: Config{
				CurrentContext: "https://missing.airbyte.com",
				Contexts: []NamedContext{
					{Name: "https://cloud.airbyte.com", Context: validContext},
				},
			},
			expectedError: "current context \"https://missing.airbyte.com\" not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := tt.config.GetCurrentContext()
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCtx, ctx)
			}
		})
	}
}

func TestConfig_SetCurrentContext(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		contextName   string
		expectedError string
	}{
		{
			name: "set existing context",
			config: Config{
				Contexts: []NamedContext{
					{Name: "context1"},
					{Name: "context2"},
				},
			},
			contextName: "context2",
		},
		{
			name: "set non-existing context",
			config: Config{
				Contexts: []NamedContext{
					{Name: "context1"},
				},
			},
			contextName:   "missing",
			expectedError: "context \"missing\" not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.SetCurrentContext(tt.contextName)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.contextName, tt.config.CurrentContext)
			}
		})
	}
}

func TestConfig_AddContext(t *testing.T) {
	newContext := Context{
		AirbyteAPIURL: "https://api.new.com",
		AirbyteURL:    "https://new.com",
		Edition:       "enterprise",
		Auth:          NewAuthWithOAuth2("new-id", "new-secret"),
	}

	tests := []struct {
		name             string
		config           Config
		contextName      string
		context          Context
		expectedContexts int
		shouldSetCurrent bool
	}{
		{
			name:             "add to empty config",
			config:           Config{},
			contextName:      "https://new.com",
			context:          newContext,
			expectedContexts: 1,
			shouldSetCurrent: true,
		},
		{
			name: "add new context",
			config: Config{
				CurrentContext: "existing",
				Contexts: []NamedContext{
					{Name: "existing"},
				},
			},
			contextName:      "https://new.com",
			context:          newContext,
			expectedContexts: 2,
			shouldSetCurrent: false,
		},
		{
			name: "update existing context",
			config: Config{
				Contexts: []NamedContext{
					{Name: "https://new.com", Context: Context{Edition: "old"}},
				},
			},
			contextName:      "https://new.com",
			context:          newContext,
			expectedContexts: 1,
			shouldSetCurrent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.AddContext(tt.contextName, tt.context)
			assert.Len(t, tt.config.Contexts, tt.expectedContexts)

			// Find the context
			found := false
			for _, nc := range tt.config.Contexts {
				if nc.Name == tt.contextName {
					found = true
					assert.Equal(t, tt.context, nc.Context)
					break
				}
			}
			assert.True(t, found)

			if tt.shouldSetCurrent {
				assert.Equal(t, tt.contextName, tt.config.CurrentContext)
			}
		})
	}
}

func TestConfig_RemoveContext(t *testing.T) {
	tests := []struct {
		name            string
		config          Config
		contextToRemove string
		expectedError   string
		expectedCurrent string
	}{
		{
			name: "remove existing context",
			config: Config{
				CurrentContext: "context1",
				Contexts: []NamedContext{
					{Name: "context1"},
					{Name: "context2"},
				},
			},
			contextToRemove: "context2",
			expectedCurrent: "context1",
		},
		{
			name: "remove current context with others",
			config: Config{
				CurrentContext: "context1",
				Contexts: []NamedContext{
					{Name: "context1"},
					{Name: "context2"},
				},
			},
			contextToRemove: "context1",
			expectedCurrent: "context2",
		},
		{
			name: "remove last context",
			config: Config{
				CurrentContext: "context1",
				Contexts: []NamedContext{
					{Name: "context1"},
				},
			},
			contextToRemove: "context1",
			expectedCurrent: "",
		},
		{
			name: "remove non-existing context",
			config: Config{
				Contexts: []NamedContext{
					{Name: "context1"},
				},
			},
			contextToRemove: "missing",
			expectedError:   "context \"missing\" not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.RemoveContext(tt.contextToRemove)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCurrent, tt.config.CurrentContext)

				// Verify context was removed
				for _, nc := range tt.config.Contexts {
					assert.NotEqual(t, tt.contextToRemove, nc.Name)
				}
			}
		})
	}
}

func TestConfig_GetCredentials(t *testing.T) {
	creds := &auth.Credentials{
		AccessToken: "test-token",
	}

	tests := []struct {
		name          string
		config        Config
		expectedCreds *auth.Credentials
		expectedError string
	}{
		{
			name: "valid credentials",
			config: Config{
				Credentials: creds,
			},
			expectedCreds: creds,
		},
		{
			name:          "no credentials",
			config:        Config{},
			expectedError: "no user credentials found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.GetCredentials()
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCreds, result)
			}
		})
	}
}

func TestConfig_IsAuthenticated(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name: "authenticated",
			config: Config{
				Credentials: &auth.Credentials{
					AccessToken: "test-token",
				},
			},
			expected: true,
		},
		{
			name:     "no credentials",
			config:   Config{},
			expected: false,
		},
		{
			name: "empty access token",
			config: Config{
				Credentials: &auth.Credentials{
					AccessToken: "",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsAuthenticated()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_SetCredentials(t *testing.T) {
	config := Config{}
	creds := &auth.Credentials{
		AccessToken: "new-token",
	}

	config.SetCredentials(creds)
	assert.Equal(t, creds, config.Credentials)
}

func TestConfig_Validate(t *testing.T) {
	validContext := Context{
		AirbyteAPIURL: "https://api.airbyte.com",
		AirbyteURL:    "https://cloud.airbyte.com",
		Edition:       "cloud",
		Auth:          NewAuthWithOAuth2("client-id", "client-secret"),
	}

	tests := []struct {
		name          string
		config        Config
		expectedError string
	}{
		{
			name: "valid config",
			config: Config{
				CurrentContext: "https://cloud.airbyte.com",
				Contexts: []NamedContext{
					{
						Name:    "https://cloud.airbyte.com",
						Context: validContext,
					},
				},
				Credentials: &auth.Credentials{
					AccessToken: "test-token",
				},
			},
		},
		{
			name: "current context not in list",
			config: Config{
				CurrentContext: "https://missing.airbyte.com",
				Contexts: []NamedContext{
					{
						Name:    "https://cloud.airbyte.com",
						Context: validContext,
					},
				},
			},
			expectedError: "current context \"https://missing.airbyte.com\" not found in contexts list",
		},
		{
			name: "empty context name",
			config: Config{
				Contexts: []NamedContext{
					{
						Name:    "",
						Context: validContext,
					},
				},
			},
			expectedError: "context name cannot be empty",
		},
		{
			name: "duplicate context names",
			config: Config{
				Contexts: []NamedContext{
					{
						Name:    "https://cloud.airbyte.com",
						Context: validContext,
					},
					{
						Name:    "https://cloud.airbyte.com",
						Context: validContext,
					},
				},
			},
			expectedError: "duplicate context name: \"https://cloud.airbyte.com\"",
		},
		{
			name: "invalid context in list",
			config: Config{
				Contexts: []NamedContext{
					{
						Name: "https://invalid.airbyte.com",
						Context: Context{
							AirbyteAPIURL: "://bad-scheme",
							AirbyteURL:    "https://cloud.airbyte.com",
							Edition:       "cloud",
							Auth:          NewAuthWithOAuth2("client-id", "client-secret"),
						},
					},
				},
			},
			expectedError: "invalid context \"https://invalid.airbyte.com\": invalid airbyteApiUrl",
		},
		{
			name: "credentials with empty access token",
			config: Config{
				CurrentContext: "https://cloud.airbyte.com",
				Contexts: []NamedContext{
					{
						Name:    "https://cloud.airbyte.com",
						Context: validContext,
					},
				},
				Credentials: &auth.Credentials{
					AccessToken: "",
				},
			},
			expectedError: "credentials present but access token is empty",
		},
		{
			name: "empty config is valid",
			config: Config{
				Contexts: []NamedContext{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
