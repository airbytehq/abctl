package cmd

import (
	"context"

	"github.com/airbytehq/abctl/internal/cmd/images"
	"github.com/airbytehq/abctl/internal/cmd/local"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/version"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
)

type verbose bool

func (v verbose) BeforeApply() error {
	pterm.EnableDebugMessages()
	return nil
}

type Cmd struct {
	Local   local.Cmd   `cmd:"" help:"Manage the local Airbyte installation."`
	Images  images.Cmd  `cmd:"" help:"Manage images used by Airbyte and abctl."`
	Version version.Cmd `cmd:"" help:"Display version information."`
	Verbose verbose     `short:"v" help:"Enable verbose output."`
}

func (c *Cmd) BeforeApply(ctx context.Context, kCtx *kong.Context) error {
	kCtx.BindTo(k8s.DefaultProvider, (*k8s.Provider)(nil))
	kCtx.BindTo(telemetry.Get(), (*telemetry.Client)(nil))

	//if err := kCtx.BindToProvider(bindK8sClient(ctx, &k8s.DefaultProvider)); err != nil {
	//	pterm.Error.Println("Unable to configure k8s client")
	//	return fmt.Errorf("unable to create k8s client: %w", err)
	//}

	return nil
}

//// bindK8sClient allows kong to make the k8s.Client injectable into a command's Run method.
//// If the cluster does exist, this will return ErrClusterNotFound.
//func bindK8sClient(ctx context.Context, provider *k8s.Provider) func() (k8s.Client, error) {
//	return func() (k8s.Client, error) {
//		k8sCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
//			&clientcmd.ClientConfigLoadingRules{ExplicitPath: provider.Kubeconfig},
//			&clientcmd.ConfigOverrides{CurrentContext: provider.Context},
//		)
//
//		if cluster, err := provider.Cluster(ctx); err != nil {
//			pterm.Error.Println("Unable to determine cluster state")
//			return nil, fmt.Errorf("unable to determine cluster state: %w", err)
//		} else if !cluster.Exists(ctx) {
//			return nil, localerr.ErrClusterNotFound
//		}
//
//		restCfg, err := k8sCfg.ClientConfig()
//		if err != nil {
//			return nil, fmt.Errorf("%w: could not create rest config: %w", localerr.ErrKubernetes, err)
//		}
//		k8sClient, err := kubernetes.NewForConfig(restCfg)
//		if err != nil {
//			return nil, fmt.Errorf("%w: could not create clientset: %w", localerr.ErrKubernetes, err)
//		}
//
//		return &k8s.DefaultK8sClient{ClientSet: k8sClient}, nil
//	}
//}
