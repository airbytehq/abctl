package helm

import (
	"fmt"
	"io"

	"github.com/airbytehq/abctl/internal/abctl"
	"github.com/airbytehq/abctl/internal/paths"
	goHelm "github.com/mittwald/go-helm-client"
	"github.com/pterm/pterm"
	"k8s.io/client-go/tools/clientcmd"
)

func ClientOptions(namespace string) *goHelm.Options {
	logger := helmLogger{}
	return &goHelm.Options{
		Namespace:        namespace,
		Output:           logger,
		DebugLog:         logger.Debug,
		Debug:            true,
		RepositoryCache:  paths.HelmRepoCache,
		RepositoryConfig: paths.HelmRepoConfig,
	}
}

// New returns the default helm client
func New(kubecfg, kubectx, namespace string) (goHelm.Client, error) {
	// Use default loading rules if kubecfg is empty (same logic as service.DefaultK8s)
	var loadingRules *clientcmd.ClientConfigLoadingRules
	if kubecfg == "" {
		// This will use KUBECONFIG env var or default ~/.kube/config
		loadingRules = clientcmd.NewDefaultClientConfigLoadingRules()
	} else {
		loadingRules = &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubecfg}
	}

	k8sCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{CurrentContext: kubectx},
	)

	restCfg, err := k8sCfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("%w: unable to create rest config: %w", abctl.ErrKubernetes, err)
	}

	helm, err := goHelm.NewClientFromRestConf(&goHelm.RestConfClientOptions{
		Options:    ClientOptions(namespace),
		RestConfig: restCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create helm client: %w", err)
	}

	return helm, nil
}

var _ io.Writer = (*helmLogger)(nil)

// helmLogger is used by the Client to convert all helm output into debug logs.
type helmLogger struct{}

func (d helmLogger) Write(p []byte) (int, error) {
	pterm.Debug.Println(fmt.Sprintf("helm: %s", string(p)))
	return len(p), nil
}

func (d helmLogger) Debug(format string, v ...interface{}) {
	pterm.Debug.Println(fmt.Sprintf("helm: "+format, v...))
}
