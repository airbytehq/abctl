package cmd

import (
	"airbyte.io/abctl/cmd/local"
	"github.com/pterm/pterm"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var (
	dnt bool

	rootCmd = &cobra.Command{
		Use:   "abctl",
		Short: pterm.LightBlue("Airbyte") + "'s proof-of-concept command-line tool",
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
	rootCmd.AddCommand(local.Cmd)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolVar(&dnt, "dnt", false, "set to opt out of telemetry data")
}
