package config

// Cmd represents the config command group
type Cmd struct {
	Init InitCmd `cmd:"" help:"Initialize abctl configuration from existing Airbyte installation."`
}