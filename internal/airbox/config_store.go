package airbox

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// EnvConfigPath is the environment variable that sets the path to the Airbox configuration.
	EnvConfigPath = "AIRBOXCONFIG"
)

// ConfigStore interface for performing config operations on the store.
type ConfigStore interface {
	Load() (*Config, error)
	Save(config *Config) error
	GetPath() string
	Exists() bool
}

// FileConfigStore implements ConfigStore using filesystem.
type FileConfigStore struct{}

// DefaultConfigPath returns the default path for airbox config.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".airbyte", "airbox", "config")
	}
	return filepath.Join(home, ".airbyte", "airbox", "config")
}

// Load and return the configuration from the file store.
func (p *FileConfigStore) Load() (*Config, error) {
	data, err := os.ReadFile(p.GetPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Save writes the configuration to the file store.
func (p *FileConfigStore) Save(config *Config) error {
	configPath := p.GetPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetPath returns configuration file path.
func (p *FileConfigStore) GetPath() string {
	if path := os.Getenv(EnvConfigPath); path != "" {
		return path
	}
	return DefaultConfigPath()
}

// Exists checks if the config file exists
func (p *FileConfigStore) Exists() bool {
	_, err := os.Stat(p.GetPath())
	return err == nil
}
