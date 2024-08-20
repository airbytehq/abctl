package cmd

import (
	"errors"
	"os"

	"github.com/airbytehq/abctl/internal/cmd/local"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/cmd/version"
	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
)

// Help messages to display for specific error situations.
const (
	// helpAirbyteDir is display if ErrAirbyteDir is ever returned
	helpAirbyteDir = `The ~/.airbyte directory is inaccessible.
You may need to remove this directory before trying your command again.`

	// helpDocker is displayed if ErrDocker is ever returned
	helpDocker = `An error occurred while communicating with the Docker daemon.
Ensure that Docker is running and is accessible.  You may need to upgrade to a newer version of Docker.
For additional help please visit https://docs.docker.com/get-docker/`

	// helpKubernetes is displayed if ErrKubernetes is ever returned
	helpKubernetes = `An error occurred while communicating with the Kubernetes cluster.
If this error persists, you may need to run the uninstall command before attempting to run
the install command again.`

	// helpIngress is displayed if ErrIngress is ever returned
	helpIngress = `An error occurred while configuring ingress.
This could be in indication that the ingress port is already in use by a different application.
The ingress port can be changed by passing the flag --port.`

	// helpPort is displayed if ErrPort is ever returned
	helpPort = `An error occurred while verifying if the request port is available.
This could be in indication that the ingress port is already in use by a different application.
The ingress port can be changed by passing the flag --port.`
)

func HandleErr(err error) {
	if err == nil {
		return
	}

	pterm.Error.Println(err)

	var errParse *kong.ParseError
	if errors.As(err, &errParse) {
		_ = kong.DefaultHelpPrinter(kong.HelpOptions{}, errParse.Context)
	}

	switch {
	case errors.Is(err, localerr.ErrAirbyteDir):
		pterm.Println()
		pterm.Info.Println(helpAirbyteDir)
	case errors.Is(err, localerr.ErrDocker):
		pterm.Println()
		pterm.Info.Println(helpDocker)
	case errors.Is(err, localerr.ErrKubernetes):
		pterm.Println()
		pterm.Info.Println(helpKubernetes)
	case errors.Is(err, localerr.ErrIngress):
		pterm.Println()
		pterm.Info.Println(helpIngress)
	case errors.Is(err, localerr.ErrPort):
		pterm.Println()
		pterm.Info.Printfln(helpPort)
	}

	os.Exit(1)
}

type verbose bool

func (v verbose) BeforeApply() error {
	pterm.EnableDebugMessages()
	return nil
}

type Cmd struct {
	Local   local.Cmd   `cmd:"" help:"Manage the local Airbyte installation."`
	Version version.Cmd `cmd:"" help:"Display version information."`
	Verbose verbose     `short:"v" help:"Enable verbose output."`
}

func (c *Cmd) BeforeApply(ctx *kong.Context) error {
	//fmt.Println("root before apply")
	if _, envVarDNT := os.LookupEnv("DO_NOT_TRACK"); envVarDNT {
		pterm.Info.Println("Telemetry collection disabled (DO_NOT_TRACK)")
	}
	ctx.BindTo(k8s.DefaultProvider, (*k8s.Provider)(nil))

	return nil
}
