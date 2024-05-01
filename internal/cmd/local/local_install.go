package local

import "github.com/spf13/cobra"

const (
	// envBasicAuthUser is the env-var that can be specified to override the default basic-auth username.
	envBasicAuthUser = "ABCTL_LOCAL_INSTALL_USERNAME"
	// envBasicAuthPass is the env-var that can be specified to override the default basic-auth password.
	envBasicAuthPass = "ABCTL_LOCAL_INSTALL_PASSWORD"
)

type InstallOptions struct {
}

func NewCmdInstall() *cobra.Command {
	var (
		flagUsername string
		flagPassword string
	)

	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Install Airbyte locally",
		PreRunE: preRunInstall,
		RunE:    runInstall,
	}

	InstallCmd.Flags().StringVarP(&flagUsername, "username", "u", "airbyte", "basic auth username, can also be specified via "+envBasicAuthUser)
	InstallCmd.Flags().StringVarP(&flagPassword, "password", "p", "password", "basic auth password, can also be specified via "+envBasicAuthPass)

	return cmd
}
