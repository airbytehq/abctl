package cmd

import (
	"airbyte.io/abctl/cmd/local"
	"github.com/pterm/pterm"
	"os"

	"github.com/spf13/cobra"
)

var (
	// flagDNT indicates if the do-not-track flag was specified
	flagDNT bool

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "abctl",
		Short: pterm.LightBlue("Airbyte") + "'s proof-of-concept command-line tool",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if flagDNT {
				pterm.Info.Println("telemetry disabled (--dnt)")
			}
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
}

func init() {
	// configure cobra to chain Persistent*Run commands together
	cobra.EnableTraverseRunHooks = true

	rootCmd.AddCommand(local.Cmd)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolVar(&flagDNT, "dnt", false, "opt out of telemetry data collection")
}
