package delete

// Cmd represents the delete command group.
type Cmd struct {
	Dataplane DataplaneCmd `cmd:"" help:"Delete a dataplane."`
}
