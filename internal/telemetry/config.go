package telemetry

import (
	"fmt"
	"github.com/oklog/ulid/v2"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const (
	welcome = `Thanks you for using Airbyte!
Anonymous usage reporting is currently enabled. For more information, please see https://docs.airbyte.com/telemetry`
)

var analyticsFile = filepath.Join(".airbyte", "analytics.yml")

// ULID is a wrapper around ulid.ULID so that we can implement the yaml interfaces.
type ULID ulid.ULID

func (u ULID) String() string {
	return ulid.ULID(u).String()
}

// UnmarshalYAML allows for converting a yaml field into a ULID.
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
	//panic("test")
	return ulid.ULID(u).String(), nil
}

type Config struct {
	UserID ULID `yaml:"anonymous_user_id"`
}

//func Load() (*Config, error) {
//	home, err := os.UserHomeDir()
//	if err != nil {
//		return nil, fmt.Errorf("could not locate home directory: %w", err)
//	}
//
//	var fullPath = filepath.Join(home, analyticsFile)
//
//	return LoadFromFile(fullPath)
//}

// permissions sets the file and directory permission level for the telemetry files that may be created.
// This is set as 0777 to match python's default mkdir behavior, as this file may be potentially shared
// between this code and PyAirbyte
const permissions = 0777

func LoadFromFile(path string) (Config, error) {
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

const header = `# This file is used by Airbyte to track anonymous usage statistics.
# For more information or to opt out, please see
# - https://docs.airbyte.com/operator-guides/telemetry
`

func WriteToFile(path string, conf Config) error {
	data, err := yaml.Marshal(conf)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	// create necessary directories
	if err := os.MkdirAll(filepath.Dir(path), permissions); err != nil {
		return fmt.Errorf("could not create directories %s: %w", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, append([]byte(header), data...), permissions); err != nil {
		return fmt.Errorf("could not write config to file %s: %w", path, err)
	}

	return nil
}
