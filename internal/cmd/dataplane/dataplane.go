package dataplane

type Cmd struct {
	Create  CreateCmd  `cmd:"" help:"Create a new dataplane."`
	Install InstallCmd `cmd:"" help:"Install Airbyte with dataplane configuration."`
}