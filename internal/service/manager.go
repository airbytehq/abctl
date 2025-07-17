package service

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/airbytehq/abctl/internal/abctl"
	"github.com/airbytehq/abctl/internal/common"
	"github.com/airbytehq/abctl/internal/docker"
	"github.com/airbytehq/abctl/internal/helm"
	"github.com/airbytehq/abctl/internal/k8s/kind"
	"github.com/airbytehq/abctl/internal/paths"
	"github.com/airbytehq/abctl/internal/pgdata"
	"k8s.io/client-go/rest"

	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/cli/browser"
	goHelm "github.com/mittwald/go-helm-client"
	"github.com/pterm/pterm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ManagerClientFactory creates and returns the Kubernetes and Helm clients
// needed by the service manager.
type ManagerClientFactory func(kubeConfig, kubeContext string) (k8s.Client, goHelm.Client, error)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// BrowserLauncher primarily for testing purposes.
type BrowserLauncher func(url string) error

// Manager for the Abctl service.
type Manager struct {
	provider k8s.Provider
	docker   *docker.Docker

	http     HTTPClient
	helm     goHelm.Client
	k8s      k8s.Client
	portHTTP int
	spinner  *pterm.SpinnerPrinter
	tel      telemetry.Client
	launcher BrowserLauncher
	userHome string
}

// Option for configuring the Manager, primarily exists for testing
type Option func(*Manager)

func WithDockerClient(client *docker.Docker) Option {
	return func(m *Manager) {
		m.docker = client
	}
}

// WithTelemetryClient define the telemetry client for this command.
func WithTelemetryClient(client telemetry.Client) Option {
	return func(m *Manager) {
		m.tel = client
	}
}

// WithHTTPClient define the http client for this command.
func WithHTTPClient(client HTTPClient) Option {
	return func(m *Manager) {
		m.http = client
	}
}

// WithHelmClient define the helm client for this command.
func WithHelmClient(client goHelm.Client) Option {
	return func(m *Manager) {
		m.helm = client
	}
}

// WithK8sClient define the k8s client for this command.
func WithK8sClient(client k8s.Client) Option {
	return func(m *Manager) {
		m.k8s = client
	}
}

// WithBrowserLauncher define the browser launcher for this command.
func WithBrowserLauncher(launcher BrowserLauncher) Option {
	return func(m *Manager) {
		m.launcher = launcher
	}
}

// WithUserHome define the user's home directory.
func WithUserHome(home string) Option {
	return func(m *Manager) {
		m.userHome = home
	}
}

func WithSpinner(spinner *pterm.SpinnerPrinter) Option {
	return func(m *Manager) {
		m.spinner = spinner
	}
}

func WithPortHTTP(port int) Option {
	return func(m *Manager) {
		m.portHTTP = port
	}
}

// NewManager initializes the service manager.
func NewManager(provider k8s.Provider, opts ...Option) (*Manager, error) {
	m := &Manager{provider: provider}
	for _, opt := range opts {
		opt(m)
	}

	// determine userhome if not defined
	if m.userHome == "" {
		m.userHome = paths.UserHome
	}

	// set http client, if not defined
	if m.http == nil {
		m.http = &http.Client{Timeout: 10 * time.Second}
	}

	if m.portHTTP == 0 {
		m.portHTTP = kind.IngressPort
	}

	// set k8s client, if not defined
	if m.k8s == nil {
		var err error
		if m.k8s, err = DefaultK8s(provider.Kubeconfig, provider.Context); err != nil {
			return nil, err
		}
	}

	// set the helm client, if not defined
	if m.helm == nil {
		var err error
		if m.helm, err = helm.New(provider.Kubeconfig, provider.Context, common.AirbyteNamespace); err != nil {
			return nil, err
		}
	}

	// set telemetry client, if not defined
	if m.tel == nil {
		m.tel = telemetry.NoopClient{}
	}

	// set spinner, if not defined
	if m.spinner == nil {
		m.spinner, _ = pterm.DefaultSpinner.Start()
	}

	// set the browser launcher, if not defined
	if m.launcher == nil {
		m.launcher = browser.OpenURL
	}

	// fetch k8s version information
	{
		k8sVersion, err := m.k8s.ServerVersionGet()
		if err != nil {
			return nil, fmt.Errorf("%w: unable to fetch kubernetes server version: %w", abctl.ErrKubernetes, err)
		}
		m.tel.Attr("k8s_version", k8sVersion)
	}

	// set provider version
	m.tel.Attr("provider", provider.Name)

	return m, nil
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
		return nil, fmt.Errorf("%w: could not create rest config: %w", abctl.ErrKubernetes, err)
	}
	k8sClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("%w: could not create clientset: %w", abctl.ErrKubernetes, err)
	}

	return &k8s.DefaultK8sClient{ClientSet: k8sClient}, nil
}

// SupportMinio checks if a MinIO persistent volume directory exists on the
// local filesystem. It returns true if the MinIO data directory exists.
// Otherwise it returns false.
func SupportMinio() (bool, error) {
	minioPath := filepath.Join(paths.Data, paths.PvMinio)
	f, err := os.Stat(minioPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to determine if minio physical volume dir exists: %w", err)
	}

	return f.IsDir(), nil
}

// EnablePsql17 checks if PostgreSQL data needs patching by examining the
// local PostgreSQL data directory. It returns true if the directory doesn't
// exist or contains PostgreSQL version 17. Otherwise it returns false.
func EnablePsql17() (bool, error) {
	pgData := pgdata.New(&pgdata.Config{
		Path: path.Join(paths.Data, paths.PvPsql, "pgdata"),
	})

	pgVersion, err := pgData.Version()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("failed to determine if any previous psql version exists: %w", err)
	}

	if pgVersion == "" || pgVersion == "17" {
		return true, nil
	}

	return false, nil
}

// DefaultManagerClientFactory initializes and returns the default Kubernetes
// and Helm clients for the service manager.
func DefaultManagerClientFactory(kubeConfig, kubeContext string) (k8s.Client, goHelm.Client, error) {
	kubeClient, err := DefaultK8s(kubeConfig, kubeContext)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize the kubernetes client: %w", err)
	}

	helmClient, err := helm.New(kubeConfig, kubeContext, common.AirbyteNamespace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize the helm client: %w", err)
	}

	return kubeClient, helmClient, nil
}
