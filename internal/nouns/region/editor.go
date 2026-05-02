package region

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
)

type Editor struct {
	Name          string `arg:"" help:"Region name."`
	MaxWorkspaces *int   `help:"New maximum number of workspaces."`
	Available     *bool  `help:"Set region availability."`
}

func (e *Editor) Run(ctx context.Context) error {
	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start(fmt.Sprintf("Updating region '%s'", e.Name))
	
	if e.MaxWorkspaces != nil {
		pterm.Info.Printfln("Setting max workspaces to %d", *e.MaxWorkspaces)
	}
	
	if e.Available != nil {
		status := "available"
		if !*e.Available {
			status = "unavailable"
		}
		pterm.Info.Printfln("Setting region to %s", status)
	}
	
	spinner.Success(fmt.Sprintf("Region '%s' updated successfully", e.Name))
	return nil
}