package telemetry

import (
	"fmt"
	"github.com/oklog/ulid/v2"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const (
	// Welcome is displayed the first time the telemetry config is created.
	Welcome = `Thanks you for using Airbyte!
Anonymous usage reporting is currently enabled. For more information, please see https://docs.airbyte.com/telemetry`
)

var ConfigFile = filepath.Join(".airbyte", "analytics.yml")

// ULID is a wrapper around ulid.ULID so that we can implement the yaml interfaces.
type ULID ulid.ULID

// NewULID returns a new randomized ULID.
func NewULID() ULID {
	return ULID(ulid.Make())
}

// String returns a string representation of this ULID.
//
//goland:noinspection GoMixedReceiverTypes
func (u ULID) String() string {
	return ulid.ULID(u).String()
}

// UnmarshalYAML allows for converting a yaml field into a ULID.
//
//goland:noinspection GoMixedReceiverTypes
func (u *ULID) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return fmt.Errorf("could not unmarshal yaml: %w", err)
	}

	parsed, err := ulid.Parse(s)
	if err != nil {
		return fmt.Errorf("could not parse ulid (%s): %w", s, err)
	}

	*u = ULID(parsed)
	return nil
}

// MarshalYAML allows for converting a ULID into a yaml field.
//
//goland:noinspection GoMixedReceiverTypes
func (u ULID) MarshalYAML() (any, error) {
	//panic("test")
	return ulid.ULID(u).String(), nil
}

// Config represents the analytics.yaml file.
type Config struct {
	UserID ULID `yaml:"anonymous_user_id"`
}

// permissions sets the file and directory permission level for the telemetry files that may be created.
// This is set as 0777 to match python's default mkdir behavior, as this file may be potentially shared
// between this code and PyAirbyte
const permissions = 0777

// loadConfigFromFile reads the file located at path and returns a Config based on it.
func loadConfigFromFile(path string) (Config, error) {
	if _, err := os.Stat(path); err != nil {
		return Config{}, fmt.Errorf("could not location file %s: %w", path, err)
	}

	analytics, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("could not read analytics file: %w", err)
	}

	var c Config

	if err := yaml.Unmarshal(analytics, &c); err != nil {
		return Config{}, fmt.Errorf("could not unmarshal yaml: %w", err)
	}

	return c, nil
}

// header is written to the start of the configuration file
const header = `# This file is used by Airbyte to track anonymous usage statistics.
# For more information or to opt out, please see
# - https://docs.airbyte.com/operator-guides/telemetry
`

// writeConfigToFile will write the cfg to the provided path.
func writeConfigToFile(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	parent := filepath.Dir(path)
	// create necessary directories
	if err := os.MkdirAll(parent, permissions); err != nil {
		return fmt.Errorf("could not create directories %s: %w", parent, err)
	}

	if err := os.WriteFile(path, append([]byte(header), data...), permissions); err != nil {
		return fmt.Errorf("could not write config to file %s: %w", path, err)
	}

	return nil
}
