package local

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	goHelm "github.com/mittwald/go-helm-client"
	"github.com/pterm/pterm"

	"github.com/airbytehq/abctl/internal/abctl"
	"github.com/airbytehq/abctl/internal/common"
	"github.com/airbytehq/abctl/internal/helm"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/paths"
	"github.com/airbytehq/abctl/internal/service"
)

type Cmd struct {
	Credentials CredentialsCmd `cmd:"" help:"Get local Airbyte user credentials."`
	Install     InstallCmd     `cmd:"" help:"Install local Airbyte."`
	Deployments DeploymentsCmd `cmd:"" help:"View local Airbyte deployments."`
	Status      StatusCmd      `cmd:"" help:"Get local Airbyte status."`
	Uninstall   UninstallCmd   `cmd:"" help:"Uninstall local Airbyte."`
}

// SvcMgrClientFactory creates and returns the Kubernetes and Helm clients
// needed by the service manager.
type SvcMgrClientFactory func(kubeConfig, kubeContext string) (k8s.Client, goHelm.Client, error)

func (c *Cmd) BeforeApply() error {
	if _, envVarDNT := os.LookupEnv("DO_NOT_TRACK"); envVarDNT {
		pterm.Info.Println("Telemetry collection disabled (DO_NOT_TRACK)")
	}

	if err := checkAirbyteDir(); err != nil {
		return fmt.Errorf("%w: %w", abctl.ErrAirbyteDir, err)
	}

	return nil
}

func (c *Cmd) AfterApply(provider k8s.Provider) error {
	pterm.Info.Println(fmt.Sprintf(
		"Using Kubernetes provider:\n  Provider: %s\n  Kubeconfig: %s\n  Context: %s",
		provider.Name, provider.Kubeconfig, provider.Context,
	))

	return nil
}

// checkAirbyteDir verifies that, if the paths.Airbyte directory exists, that it has proper permissions.
// If the directory does not have the proper permissions, this method will attempt to fix them.
// A nil response either indicates that either:
// - no paths.Airbyte directory exists
// - the permissions are already correct
// - this function was able to fix the incorrect permissions.
func checkAirbyteDir() error {
	fileInfo, err := os.Stat(paths.Airbyte)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// nothing to do, directory will be created later on
			return nil
		}
		return fmt.Errorf("unable to determine status of '%s': %w", paths.Airbyte, err)
	}

	if !fileInfo.IsDir() {
		return errors.New(paths.Airbyte + " is not a directory")
	}

	if fileInfo.Mode().Perm() >= 0744 {
		// directory has minimal permissions
		return nil
	}

	if err := os.Chmod(paths.Airbyte, 0744); err != nil {
		return fmt.Errorf("unable to change permissions of '%s': %w", paths.Airbyte, err)
	}

	return nil
}

// DefaultSvcMgrClientFactory initializes and returns the default Kubernetes
// and Helm clients for the service manager.
func DefaultSvcMgrClientFactory(kubeConfig, kubeContext string) (k8s.Client, goHelm.Client, error) {
	kubeClient, err := service.DefaultK8s(kubeConfig, kubeContext)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize the kubernetes client: %w", err)
	}

	helmClient, err := helm.New(kubeConfig, kubeContext, common.AirbyteNamespace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize the helm client: %w", err)
	}

	return kubeClient, helmClient, nil
}
