package cmd

import (
	"fmt"
	"os"

	"github.com/airbytehq/abctl/internal/cmd/local"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/cmd/version"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

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
	if _, envVarDNT := os.LookupEnv("DO_NOT_TRACK"); envVarDNT {
		pterm.Info.Println("Telemetry collection disabled (DO_NOT_TRACK)")
	}
	ctx.BindTo(k8s.DefaultProvider, (*k8s.Provider)(nil))
	ctx.BindTo(telemetry.Get(), (*telemetry.Client)(nil))
	if err := ctx.BindToProvider(bindK8sClient(&k8s.DefaultProvider)); err != nil {
		pterm.Error.Println("Unable to configure k8s client")
		return fmt.Errorf("unable to create k8s client: %w", err)
	}

	return nil
}

// bindK8sClient allows kong to make the k8s.Client injectable into a command's Run method.
// If the cluster does exist, this will return ErrClusterNotFound.
func bindK8sClient(provider *k8s.Provider) func() (k8s.Client, error) {
	return func() (k8s.Client, error) {
		k8sCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: provider.Kubeconfig},
			&clientcmd.ConfigOverrides{CurrentContext: provider.Context},
		)

		if cluster, err := provider.Cluster(); err != nil {
			pterm.Error.Println("Unable to determine cluster state")
			return nil, fmt.Errorf("unable to determine cluster state: %w", err)
		} else if !cluster.Exists() {
			return nil, localerr.ErrClusterNotFound
		}

		restCfg, err := k8sCfg.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("%w: could not create rest config: %w", localerr.ErrKubernetes, err)
		}
		k8sClient, err := kubernetes.NewForConfig(restCfg)
		if err != nil {
			return nil, fmt.Errorf("%w: could not create clientset: %w", localerr.ErrKubernetes, err)
		}

		return &k8s.DefaultK8sClient{ClientSet: k8sClient}, nil
	}
}
