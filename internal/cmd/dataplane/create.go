package dataplane

import (
	"github.com/pterm/pterm"
)

type CreateCmd struct {
}

func (c *CreateCmd) Run() error {
	pterm.Success.Println("Dataplane create command executed successfully")
	return nil
}