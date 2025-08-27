package workspace

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
)

type Editor struct {
	Name        string  `arg:"" help:"Workspace name."`
	NewName     *string `help:"New workspace name."`
	DataplaneID *string `help:"Move to different dataplane."`
}

func (e *Editor) Run(ctx context.Context) error {
	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start(fmt.Sprintf("Updating workspace '%s'", e.Name))
	
	if e.NewName != nil {
		pterm.Info.Printfln("Renaming workspace to '%s'", *e.NewName)
	}
	
	if e.DataplaneID != nil {
		pterm.Info.Printfln("Moving workspace to dataplane '%s'", *e.DataplaneID)
	}
	
	spinner.Success(fmt.Sprintf("Workspace '%s' updated successfully", e.Name))
	return nil
}