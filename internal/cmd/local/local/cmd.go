package local

import (
	"fmt"
	"net/http"
	"time"

	"github.com/airbytehq/abctl/internal/cmd/local/helm"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s/kind"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"k8s.io/client-go/rest"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/cli/browser"
	"github.com/pterm/pterm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	airbyteChartName    = "airbyte/airbyte"
	airbyteChartRelease = "airbyte-abctl"
	airbyteIngress      = "ingress-abctl"
	airbyteNamespace    = "airbyte-abctl"
	airbyteRepoName     = "airbyte"
	airbyteRepoURL      = "https://airbytehq.github.io/helm-charts"
	nginxChartName      = "nginx/ingress-nginx"
	nginxChartRelease   = "ingress-nginx"
	nginxNamespace      = "ingress-nginx"
	nginxRepoName       = "nginx"
	nginxRepoURL        = "https://kubernetes.github.io/ingress-nginx"
)

// dockerAuthSecretName is the name of the secret which holds the docker authentication information.
const dockerAuthSecretName = "docker-auth"

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// BrowserLauncher primarily for testing purposes.
type BrowserLauncher func(url string) error

// ChartLocator primarily for testing purposes.
type ChartLocator func(repoName, repoUrl string) string

// Installer is the local command, responsible for installing, uninstalling, or other local actions.
type Installer struct {
	provider    k8s.Provider
	http        HTTPClient
	helm        helm.Client
	k8s         k8s.Client
	portHTTP    int
	spinner     *pterm.SpinnerPrinter
	tel         telemetry.Client
	launcher    BrowserLauncher
	locateChart ChartLocator
	userHome    string
}

// Option for configuring the Command, primarily exists for testing
type Option func(*Installer)

// WithTelemetryClient define the telemetry client for this command.
func WithTelemetryClient(client telemetry.Client) Option {
	return func(c *Installer) {
		c.tel = client
	}
}

// WithHTTPClient define the http client for this command.
func WithHTTPClient(client HTTPClient) Option {
	return func(c *Installer) {
		c.http = client
	}
}

// WithHelmClient define the helm client for this command.
func WithHelmClient(client helm.Client) Option {
	return func(c *Installer) {
		c.helm = client
	}
}

// WithK8sClient define the k8s client for this command.
func WithK8sClient(client k8s.Client) Option {
	return func(c *Installer) {
		c.k8s = client
	}
}

// WithBrowserLauncher define the browser launcher for this command.
func WithBrowserLauncher(launcher BrowserLauncher) Option {
	return func(c *Installer) {
		c.launcher = launcher
	}
}

func WithChartLocator(locator ChartLocator) Option {
	return func(c *Installer) {
		c.locateChart = locator
	}
}

// WithUserHome define the user's home directory.
func WithUserHome(home string) Option {
	return func(c *Installer) {
		c.userHome = home
	}
}

func WithSpinner(spinner *pterm.SpinnerPrinter) Option {
	return func(c *Installer) {
		c.spinner = spinner
	}
}

func WithPortHTTP(port int) Option {
	return func(c *Installer) {
		c.portHTTP = port
	}
}

// NewInstaller creates a new Installer
func NewInstaller(provider k8s.Provider, opts ...Option) (*Installer, error) {
	c := &Installer{provider: provider}
	for _, opt := range opts {
		opt(c)
	}

	if c.locateChart == nil {
		c.locateChart = locateLatestAirbyteChart
	}

	// determine userhome if not defined
	if c.userHome == "" {
		c.userHome = paths.UserHome
	}

	// set http client, if not defined
	if c.http == nil {
		c.http = &http.Client{Timeout: 10 * time.Second}
	}

	if c.portHTTP == 0 {
		c.portHTTP = kind.IngressPort
	}

	// set k8s client, if not defined
	if c.k8s == nil {
		var err error
		if c.k8s, err = defaultK8s(provider.Kubeconfig, provider.Context); err != nil {
			return nil, err
		}
	}

	// set the helm client, if not defined
	if c.helm == nil {
		var err error
		if c.helm, err = helm.New(provider.Kubeconfig, provider.Context, airbyteNamespace); err != nil {
			return nil, err
		}
	}

	// set telemetry client, if not defined
	if c.tel == nil {
		c.tel = telemetry.NoopClient{}
	}

	// set spinner, if not defined
	if c.spinner == nil {
		c.spinner, _ = pterm.DefaultSpinner.Start()
	}

	// set the browser launcher, if not defined
	if c.launcher == nil {
		c.launcher = browser.OpenURL
	}

	// fetch k8s version information
	{
		k8sVersion, err := c.k8s.ServerVersionGet()
		if err != nil {
			return nil, fmt.Errorf("%w: unable to fetch kubernetes server version: %w", localerr.ErrKubernetes, err)
		}
		c.tel.Attr("k8s_version", k8sVersion)
	}

	// set provider version
	c.tel.Attr("provider", provider.Name)

	return c, nil
}

// defaultK8s returns the default k8s client
func defaultK8s(kubecfg, kubectx string) (k8s.Client, error) {
	rest.SetDefaultWarningHandler(k8s.Logger{})
	k8sCfg, err := k8sClientConfig(kubecfg, kubectx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", localerr.ErrKubernetes, err)
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

// k8sClientConfig returns a k8s client config using the ~/.kube/config file and the k8sContext context.
func k8sClientConfig(kubecfg, kubectx string) (clientcmd.ClientConfig, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubecfg},
		&clientcmd.ConfigOverrides{CurrentContext: kubectx},
	), nil
}
