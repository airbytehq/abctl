package helm

import (
	"context"
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/status"
	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"io"
	"k8s.io/client-go/tools/clientcmd"
)

// Client primarily for testing purposes
type Client interface {
	AddOrUpdateChartRepo(entry repo.Entry) error
	GetChart(name string, options *action.ChartPathOptions) (*chart.Chart, string, error)
	GetRelease(name string) (*release.Release, error)
	InstallOrUpgradeChart(ctx context.Context, spec *helmclient.ChartSpec, opts *helmclient.GenericHelmOptions) (*release.Release, error)
	UninstallReleaseByName(name string) error
}

// New returns the default helm client
func New(kubecfg, kubectx, namespace string) (Client, error) {
	k8sCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubecfg},
		&clientcmd.ConfigOverrides{CurrentContext: kubectx},
	)

	restCfg, err := k8sCfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("%w: unable to create rest config: %w", localerr.ErrKubernetes, err)
	}

	logger := helmLogger{}
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

var _ io.Writer = (*helmLogger)(nil)

// helmLogger is used by the Client to convert all helm output into debug logs.
type helmLogger struct {
}

func (d helmLogger) Write(p []byte) (int, error) {
	status.Debug(fmt.Sprintf("helm: %s", string(p)))
	return len(p), nil
}

func (d helmLogger) Debug(format string, v ...interface{}) {
	status.Debug(fmt.Sprintf("helm - DEBUG: "+format, v...))
}
