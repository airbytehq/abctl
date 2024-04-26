package local

import (
	"github.com/airbytehq/abctl/internal/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/spf13/cobra"
)

// telClient is the telemetry telClient to use.
// This will be set in the persistentPreRunLocal method which runs prior to any commands being executed.
var telClient telemetry.Client

// provider is which provider is being used.
// This will be set in the persistentPreRunLocal method which runs prior to any commands being executed.
var provider k8s.Provider

// Cmd represents the local command
var Cmd = &cobra.Command{
	Use:               "local",
	PersistentPreRunE: persistentPreRunLocal,
	Short:             "Manages local Airbyte installations",
}

const (
	// envBasicAuthUser is the env-var that can be specified to override the default basic-auth username.
	envBasicAuthUser = "ABCTL_LOCAL_INSTALL_USERNAME"
	// envBasicAuthPass is the env-var that can be specified to override the default basic-auth password.
	envBasicAuthPass = "ABCTL_LOCAL_INSTALL_PASSWORD"
)

const Port = 9899

// InstallCmd installs Airbyte locally
var InstallCmd = &cobra.Command{
	Use:     "install",
	Short:   "Install Airbyte locally",
	PreRunE: preRunInstall,
	RunE:    runInstall,
}

// UninstallCmd uninstalls Airbyte locally
var UninstallCmd = &cobra.Command{
	Use:     "uninstall",
	Short:   "Uninstall Airbyte locally",
	PreRunE: preRunUninstall,
	RunE:    runUninstall,
}

var (
	flagUsername    string
	flagPassword    string
	flagProvider    string
	flagKubeconfig  string
	flagKubeContext string
	flagPort        int
)

func init() {
	InstallCmd.Flags().StringVarP(&flagUsername, "username", "u", "airbyte", "basic auth username, can also be specified via "+envBasicAuthUser)
	InstallCmd.Flags().StringVarP(&flagPassword, "password", "p", "password", "basic auth password, can also be specified via "+envBasicAuthPass)

	Cmd.PersistentFlags().StringVarP(&flagProvider, "k8s-provider", "k", k8s.KindProvider.Name, "kubernetes provider to use")
	Cmd.PersistentFlags().StringVarP(&flagKubeconfig, "kubeconfig", "", "", "kubernetes config file to use")
	Cmd.PersistentFlags().StringVarP(&flagKubeContext, "kubecontext", "", "", "kubernetes context to use")
	Cmd.PersistentFlags().IntVarP(&flagPort, "port", "", Port, "ingress http port")

	Cmd.AddCommand(InstallCmd)
	Cmd.AddCommand(UninstallCmd)
}
