package version

import (
	"fmt"
	"strings"

	"github.com/airbytehq/abctl/internal/build"
	"github.com/pterm/pterm"
)

type Cmd struct{}

func (c *Cmd) Run() error {
	parts := []string{fmt.Sprintf("version: %s", build.Version)}
	if build.Revision != "" {
		parts = append(parts, fmt.Sprintf("revision: %s", build.Revision))
	}
	if build.ModificationTime != "" {
		parts = append(parts, fmt.Sprintf("time: %s", build.ModificationTime))
	}
	if build.Modified {
		parts = append(parts, fmt.Sprintf("modified: %t", build.Modified))
	}
	pterm.Println(strings.Join(parts, "\n"))

	return nil
}
