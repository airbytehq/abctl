package cmd

import (
	"context"

	"github.com/airbytehq/abctl/internal/cmd/images"
	"github.com/airbytehq/abctl/internal/cmd/local"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/version"
	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
)

type verbose bool

func (v verbose) BeforeApply() error {
	pterm.EnableDebugMessages()
	return nil
}

type Cmd struct {
	Local   local.Cmd   `cmd:"" help:"Manage the local Airbyte installation."`
	Images  images.Cmd  `cmd:"" help:"Manage images used by Airbyte and abctl."`
	Version version.Cmd `cmd:"" help:"Display version information."`
	Verbose verbose     `short:"v" help:"Enable verbose output."`
}

func (c *Cmd) BeforeApply(ctx context.Context, kCtx *kong.Context) error {
	kCtx.BindTo(k8s.DefaultProvider, (*k8s.Provider)(nil))
	return nil
}
