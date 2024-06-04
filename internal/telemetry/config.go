package telemetry

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const (
	// Welcome is displayed the first time the telemetry config is created.
	Welcome = `Thanks for using Airbyte!
Anonymous usage reporting is currently enabled. For more information, please see https://docs.airbyte.com/telemetry`
)

var ConfigFile = filepath.Join(".airbyte", "analytics.yml")

// UUID is a wrapper around uuid.UUID so that we can implement the yaml interfaces.
type UUID uuid.UUID

// NewUUID returns a new randomized UUID.
func NewUUID() UUID {
	return UUID(uuid.New())
}

// String returns a string representation of this UUID.
func (u UUID) String() string {
	return u.toUUID().String()
}

func (u *UUID) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return fmt.Errorf("could not unmarshal yaml: %w", err)
	}

	parsed, err := uuid.Parse(s)
	if err != nil {
		return fmt.Errorf("could not parse uuid (%s): %w", s, err)
	}

	*u = UUID(parsed)
	return nil
}

func (u UUID) MarshalYAML() (any, error) {
	return u.toUUID().String(), nil
}

// IsZero implements the yaml interface, used to treat a uuid.Nil as empty for yaml purposes
func (u UUID) IsZero() bool {
	return u.String() == uuid.Nil.String()
}

// toUUID converts the telemetry.UUID type back to the underlying uuid.UUID type
func (u UUID) toUUID() uuid.UUID {
	return uuid.UUID(u)
}

// ULID is a wrapper around ulid.ULID so that we can implement the yaml interfaces.
// Deprecated: use UUID instead
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
func (u ULID) MarshalYAML() (any, error) {
	return ulid.ULID(u).String(), nil
}

func (u ULID) IsZero() bool {
	return u.String() == "00000000000000000000000000"
}

// Config represents the analytics.yaml file.
type Config struct {
	UserID   ULID `yaml:"anonymous_user_id,omitempty"`
	UserUUID UUID `yaml:"analytics_id,omitempty"`
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
