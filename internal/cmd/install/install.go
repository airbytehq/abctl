package install

// Cmd represents the install command group
type Cmd struct {
	Dataplane DataplaneCmd `cmd:"" help:"Install a new dataplane."`
}
