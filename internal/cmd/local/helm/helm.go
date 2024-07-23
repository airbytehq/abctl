package helm

import (
	"context"
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/tools/clientcmd"
)

// Client primarily for testing purposes
type Client interface {
	AddOrUpdateChartRepo(entry repo.Entry) error
	GetChart(string, *action.ChartPathOptions) (*chart.Chart, string, error)
	GetRelease(name string) (*release.Release, error)
	InstallOrUpgradeChart(ctx context.Context, spec *helmclient.ChartSpec, opts *helmclient.GenericHelmOptions) (*release.Release, error)
	UninstallReleaseByName(string) error
}

// New returns the default helm client
func New(kubecfg, kubectx, namespace string) (Client, error) {
	k8sCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubecfg},
		&clientcmd.ConfigOverrides{CurrentContext: kubectx},
	)

	restCfg, err := k8sCfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("%w: could not create rest config: %w", localerr.ErrKubernetes, err)
	}

	logger := &helmLogger{}
	helm, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace: namespace,
			Output:    logger,
			DebugLog:  logger.Debug,
			Debug:     true,
		},
		RestConfig: restCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create helm client: %w", err)
	}

	return helm, nil
}

// helmLogger is used by the Client to convert all helm output into debug logs.
type helmLogger struct {
}

func (d *helmLogger) Write(p []byte) (int, error) {
	pterm.Debug.Println(fmt.Sprintf("helm: %s", string(p)))
	return len(p), nil
}

func (d *helmLogger) Debug(format string, v ...interface{}) {
	pterm.Debug.Println(fmt.Sprintf("helm - DEBUG: "+format, v...))
}
