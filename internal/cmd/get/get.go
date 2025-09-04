package get

// Cmd represents the get command group.
type Cmd struct {
	Dataplane DataplaneCmd `cmd:"" aliases:"dataplanes" help:"Get dataplane details."`
}
