package dataplane

import (
	"github.com/pterm/pterm"
)

type Cmd struct {
}

func (c *Cmd) Run() error {
	pterm.Success.Println("Dataplane command executed successfully")
	return nil
}