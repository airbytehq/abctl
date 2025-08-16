package dataplane

type Cmd struct {
	Create CreateCmd `cmd:"" help:"Create a new dataplane."`
}