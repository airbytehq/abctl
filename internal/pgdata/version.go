package pgdata

import (
	"fmt"
	"os"
	"path"
	"strings"
)

// Client for performing PGDATA actions.
type Client struct {
	cfg Config
}

// Config stores the config needed to interact with PGDATA.
type Config struct {
	Path string
}

// New returns an initialized PGDATA client.
func New(cfg *Config) *Client {
	return &Client{
		cfg: *cfg,
	}
}

// Version is returned for the PGDATA dir.
func (c *Client) Version() (string, error) {
	versionFile := path.Join(c.cfg.Path, "PG_VERSION")
	b, err := os.ReadFile(versionFile)
	if err != nil {
		return "", fmt.Errorf("error reading pgdata version file: %w", err)
	}

	return strings.TrimRight(string(b), "\n"), nil
}
