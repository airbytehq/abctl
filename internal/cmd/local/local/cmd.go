package local

import (
	"fmt"
	"net/http"
	"time"

	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/cmd/local/helm"
	"github.com/airbytehq/abctl/internal/cmd/local/k8s/kind"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/airbytehq/abctl/internal/common"
	"k8s.io/client-go/rest"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/cli/browser"
	"github.com/pterm/pterm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// BrowserLauncher primarily for testing purposes.
type BrowserLauncher func(url string) error

// Command is the local command, responsible for installing, uninstalling, or other local actions.
type Command struct {
	provider k8s.Provider
	docker   *docker.Docker

	http     HTTPClient
	helm     helm.Client
	k8s      k8s.Client
	portHTTP int
	spinner  *pterm.SpinnerPrinter
	tel      telemetry.Client
	launcher BrowserLauncher
	userHome string
}

// Option for configuring the Command, primarily exists for testing
type Option func(*Command)

func WithDockerClient(client *docker.Docker) Option {
	return func(c *Command) {
		c.docker = client
	}
}

// WithTelemetryClient define the telemetry client for this command.
func WithTelemetryClient(client telemetry.Client) Option {
	return func(c *Command) {
		c.tel = client
	}
}

// WithHTTPClient define the http client for this command.
func WithHTTPClient(client HTTPClient) Option {
	return func(c *Command) {
		c.http = client
	}
}

// WithHelmClient define the helm client for this command.
func WithHelmClient(client helm.Client) Option {
	return func(c *Command) {
		c.helm = client
	}
}

// WithK8sClient define the k8s client for this command.
func WithK8sClient(client k8s.Client) Option {
	return func(c *Command) {
		c.k8s = client
	}
}

// WithBrowserLauncher define the browser launcher for this command.
func WithBrowserLauncher(launcher BrowserLauncher) Option {
	return func(c *Command) {
		c.launcher = launcher
	}
}

// WithUserHome define the user's home directory.
func WithUserHome(home string) Option {
	return func(c *Command) {
		c.userHome = home
	}
}

func WithSpinner(spinner *pterm.SpinnerPrinter) Option {
	return func(c *Command) {
		c.spinner = spinner
	}
}

func WithPortHTTP(port int) Option {
	return func(c *Command) {
		c.portHTTP = port
	}
}

// New creates a new Command
func New(provider k8s.Provider, opts ...Option) (*Command, error) {
	c := &Command{provider: provider}
	for _, opt := range opts {
		opt(c)
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
		if c.k8s, err = DefaultK8s(provider.Kubeconfig, provider.Context); err != nil {
			return nil, err
		}
	}

	// set the helm client, if not defined
	if c.helm == nil {
		var err error
		if c.helm, err = helm.New(provider.Kubeconfig, provider.Context, common.AirbyteNamespace); err != nil {
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

// DefaultK8s returns the default k8s client
func DefaultK8s(kubecfg, kubectx string) (k8s.Client, error) {
	rest.SetDefaultWarningHandler(k8s.Logger{})
	k8sCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubecfg},
		&clientcmd.ConfigOverrides{CurrentContext: kubectx},
	)

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
