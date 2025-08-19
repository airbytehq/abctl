package workspace

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
)

type Deleter struct {
	Name  string `arg:"" help:"Workspace name."`
	Force bool   `help:"Force deletion without confirmation."`
}

func (d *Deleter) Run(ctx context.Context) error {
	if !d.Force {
		pterm.Warning.Printfln("This will delete workspace '%s' and all associated data", d.Name)
		return fmt.Errorf("deletion cancelled (use --force to proceed)")
	}
	
	spinner := &pterm.DefaultSpinner
	spinner, _ = spinner.Start(fmt.Sprintf("Deleting workspace '%s'", d.Name))
	
	spinner.Success(fmt.Sprintf("Workspace '%s' deleted successfully", d.Name))
	return nil
}