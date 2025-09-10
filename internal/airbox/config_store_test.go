package airbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/airbytehq/abctl/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileConfigStore_GetPath(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		expectValue string
		expectParts []string
	}{
		{
			name:        "uses env var when set",
			envValue:    "/custom/path/config.yaml",
			expectValue: "/custom/path/config.yaml",
		},
		{
			name:        "uses default path when env not set",
			envValue:    "",
			expectParts: []string{".airbyte", "airbox", "config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				_ = os.Setenv(EnvConfigPath, tt.envValue)
			} else {
				_ = os.Unsetenv(EnvConfigPath)
			}
			t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

			store := &FileConfigStore{}
			path := store.GetPath()

			if tt.expectValue != "" {
				assert.Equal(t, tt.expectValue, path)
			} else {
				for _, part := range tt.expectParts {
					assert.Contains(t, path, part)
				}
			}
		})
	}
}

func TestFileConfigStore_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	_ = os.Setenv(EnvConfigPath, configPath)
	t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

	store := &FileConfigStore{}

	// Test config
	testConfig := &Config{
		CurrentContext: "https://test.airbyte.com",
		Contexts: []NamedContext{
			{
				Name: "https://test.airbyte.com",
				Context: Context{
					AirbyteAPIURL:  "https://api.test.airbyte.com",
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

	// Test Save
	err := store.Save(testConfig)
	require.NoError(t, err)

	// Verify file exists
	assert.True(t, store.Exists())

	// Test Load
	loadedConfig, err := store.Load()
	require.NoError(t, err)

	// Verify loaded config
	assert.Equal(t, testConfig.CurrentContext, loadedConfig.CurrentContext)
	assert.Len(t, loadedConfig.Contexts, 1)
	assert.Equal(t, testConfig.Contexts[0].Name, loadedConfig.Contexts[0].Name)
}

func TestFileConfigStore_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	_ = os.Setenv(EnvConfigPath, configPath)
	t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

	store := &FileConfigStore{}

	// File doesn't exist initially
	assert.False(t, store.Exists())

	// Create file
	testConfig := &Config{
		CurrentContext: "test",
		Contexts:       []NamedContext{},
	}
	err := store.Save(testConfig)
	require.NoError(t, err)

	// File should exist now
	assert.True(t, store.Exists())
}

func TestFileConfigStore_LoadErrors(t *testing.T) {
	tests := []struct {
		name          string
		setupFile     func(t *testing.T) string
		expectedError string
	}{
		{
			name: "file does not exist",
			setupFile: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			expectedError: "failed to read config file",
		},
		{
			name: "invalid YAML",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "invalid.yaml")
				_ = os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0o600)
				return configPath
			},
			expectedError: "failed to parse config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupFile(t)
			_ = os.Setenv(EnvConfigPath, configPath)
			t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

			store := &FileConfigStore{}
			config, err := store.Load()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			assert.Nil(t, config)
		})
	}
}

func TestFileConfigStore_SaveErrors(t *testing.T) {
	tests := []struct {
		name          string
		setupPath     func(t *testing.T) string
		expectedError string
	}{
		{
			name: "directory creation fails",
			setupPath: func(t *testing.T) string {
				// Create a file where we want a directory
				tmpDir := t.TempDir()
				blockingFile := filepath.Join(tmpDir, "blocking")
				_ = os.WriteFile(blockingFile, []byte("content"), 0o644)
				return filepath.Join(blockingFile, "config.yaml")
			},
			expectedError: "failed to create config directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupPath(t)
			_ = os.Setenv(EnvConfigPath, configPath)
			t.Cleanup(func() { _ = os.Unsetenv(EnvConfigPath) })

			store := &FileConfigStore{}
			config := &Config{
				CurrentContext: "test",
				Contexts: []NamedContext{
					{Name: "test", Context: Context{AirbyteURL: "https://test.com"}},
				},
			}

			err := store.Save(config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	assert.Contains(t, path, ".airbyte")
	assert.Contains(t, path, "airbox")
	assert.Contains(t, path, "config")
}
