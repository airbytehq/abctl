package cmd

import (
	"context"
	"errors"
	"github.com/airbytehq/abctl/internal/cmd/local"
	"github.com/airbytehq/abctl/internal/cmd/version"
	"github.com/airbytehq/abctl/internal/local/localerr"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
)

// Help messages to display for specific error situations.
const (
	// helpDocker is displayed if ErrDocker is ever returned
	helpDocker = `An error occurred while communicating with the Docker daemon.
Ensure that Docker is running and is accessible.  You may need to upgrade to a newer version of Docker.
For additional help please visit https://docs.docker.com/get-docker/`

	// helpKubernetes is displayed if ErrKubernetes is ever returned
	helpKubernetes = `An error occurred while communicating with the Kubernetes cluster.
If using Docker Desktop, ensure that Kubernetes is enabled.
For additional help please visit https://docs.docker.com/desktop/kubernetes/`

	// helpIngress is displayed if ErrIngress is ever returned
	helpIngress = `An error occurred while configuring ingress.
This could be in indication that the ingress port is already in use by a different application.
The ingress port can be changed by passing the flag --port.`

	// helpPort is displayed if ErrPort is ever returned
	helpPort = `An error occurred while verifying if the request port is available.
This could be in indication that the ingress port is already in use by a different application.
The ingress port can be changed by passing the flag --port.`
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context, cmd *cobra.Command) {
	if err := cmd.ExecuteContext(ctx); err != nil {
		pterm.Error.Println(err)

		if errors.Is(err, localerr.ErrDocker) {
			pterm.Println()
			pterm.Info.Println(helpDocker)
		} else if errors.Is(err, localerr.ErrKubernetes) {
			pterm.Println()
			pterm.Info.Println(helpKubernetes)
		} else if errors.Is(err, localerr.ErrIngress) {
			pterm.Println()
			pterm.Info.Println(helpIngress)
		} else if errors.Is(err, localerr.ErrPort) {
			pterm.Println()
			pterm.Info.Printfln(helpPort)
		}

		os.Exit(1)
	}
}

// NewCmd returns the abctl root cobra command.
func NewCmd() *cobra.Command {
	cobra.EnableTraverseRunHooks = true

	var (
		flagDNT     bool
		flagVerbose bool
	)

	cmd := &cobra.Command{
		Use:   "abctl",
		Short: pterm.LightBlue("Airbyte") + "'s command line tool",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if flagVerbose {
				pterm.EnableDebugMessages()
			}

			if flagDNT {
				pterm.Info.Println("Telemetry collection disabled (--dnt)")
			}

			return nil
		},
	}

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.CompletionOptions.DisableDefaultCmd = true

	cmd.PersistentFlags().BoolVar(&flagDNT, "dnt", false, "opt out of telemetry data collection")
	cmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "enable verbose output")

	cmd.AddCommand(version.NewCmdVersion())
	cmd.AddCommand(local.NewCmdLocal())

	return cmd
}
