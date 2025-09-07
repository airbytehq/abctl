package cmd

import (
	"context"

	"github.com/airbytehq/abctl/internal/cmd/verbs"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
)

type verboseNew bool

func (v verboseNew) BeforeApply() error {
	pterm.EnableDebugMessages()
	return nil
}

type CmdNew struct {
	Get    verbs.GetCmd    `cmd:"" help:"Get resource information."`
	List   verbs.ListCmd   `cmd:"" help:"List resources."`
	Create verbs.CreateCmd `cmd:"" help:"Create resources."`
	Edit   verbs.EditCmd   `cmd:"" help:"Edit resources."`
	Delete verbs.DeleteCmd `cmd:"" help:"Delete resources."`
	Logs   verbs.LogsCmd   `cmd:"" help:"View resource logs."`

	Verbose verboseNew `short:"v" help:"Enable verbose output."`
}

func (c *CmdNew) BeforeApply(_ context.Context, kCtx *kong.Context) error {
	kCtx.BindTo(k8s.DefaultProvider, (*k8s.Provider)(nil))
	kCtx.BindTo(service.DefaultManagerClientFactory, (*service.ManagerClientFactory)(nil))
	return nil
}