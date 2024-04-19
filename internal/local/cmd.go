package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/airbytehq/abctl/internal/local/k8s"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/pterm/pterm"
	"golang.org/x/crypto/bcrypt"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	airbyteChartName    = "airbyte/airbyte"
	airbyteChartRelease = "airbyte-abctl"
	airbyteIngress      = "ingress-abctl"
	airbyteNamespace    = "abctl"
	airbyteRepoName     = "airbyte"
	airbyteRepoURL      = "https://airbytehq.github.io/helm-charts"
	clusterName         = "airbyte-abctl"
	clusterPort         = 6162
	nginxChartName      = "nginx/ingress-nginx"
	nginxChartRelease   = "ingress-nginx"
	nginxNamespace      = "ingress-nginx"
	nginxRepoName       = "nginx"
	nginxRepoURL        = "https://kubernetes.github.io/ingress-nginx"
	k8sContext          = "docker-desktop"
)

// HelmClient primarily for testing purposes
type HelmClient interface {
	AddOrUpdateChartRepo(entry repo.Entry) error
	GetChart(string, *action.ChartPathOptions) (*chart.Chart, string, error)
	GetRelease(name string) (*release.Release, error)
	InstallOrUpgradeChart(ctx context.Context, spec *helmclient.ChartSpec, opts *helmclient.GenericHelmOptions) (*release.Release, error)
	UninstallReleaseByName(string) error
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// BrowserLauncher for
type BrowserLauncher func(url string) error

// ErrDocker is returned anytime an error specific to docker occurs.
var ErrDocker = errors.New("error communicating with docker")

// DockerClient primarily for testing purposes
type DockerClient interface {
	ServerVersion(context.Context) (types.Version, error)
}

// Command is the local command, responsible for installing, uninstalling, or other local actions.
type Command struct {
	cluster  k8s.Cluster
	docker   DockerClient
	http     HTTPClient
	helm     HelmClient
	k8s      k8s.K8sClient
	tel      telemetry.Client
	launcher BrowserLauncher
	userHome string
}

// Option for configuring the Command, primarily exists for testing
type Option func(*Command)

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
func WithHelmClient(client HelmClient) Option {
	return func(c *Command) {
		c.helm = client
	}
}

// WithK8sClient define the k8s client for this command.
func WithK8sClient(client k8s.K8sClient) Option {
	return func(c *Command) {
		c.k8s = client
	}
}

// WithDockerClient define the docker client for this command.
func WithDockerClient(client DockerClient) Option {
	return func(c *Command) {
		c.docker = client
	}
}

// WithBrowserLauncher define the browser launcher for this command.
func WithBrowserLauncher(launcher BrowserLauncher) Option {
	return func(c *Command) {
		c.launcher = launcher
	}
}

// New creates a new Command
func New(provider k8s.Provider, opts ...Option) (*Command, error) {
	c := &Command{}
	for _, opt := range opts {
		opt(c)
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine user home directory: %w", err)
	}
	c.userHome = userHome

	// set docker client if not defined
	if c.docker == nil {
		if c.docker, err = defaultDocker(c.userHome); err != nil {
			return nil, err
		}
	}

	// set http client, if not defined
	if c.http == nil {
		c.http = &http.Client{Timeout: 10 * time.Second}
	}

	// set k8s client, if not defined
	if c.k8s == nil {
		if c.k8s, err = defaultK8s(c.userHome); err != nil {
			return nil, err
		}
	}

	// set the helm client, if not defined
	if c.helm == nil {
		if c.helm, err = defaultHelm(c.userHome); err != nil {
			return nil, err
		}
	}

	// set telemetry client, if not defined
	if c.tel == nil {
		c.tel = telemetry.NoopClient{}
	}

	if c.launcher == nil {
		c.launcher = func(url string) error {
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				cmd = exec.Command("open", url)
			case "windows":
				cmd = exec.Command("cmd", "/c", "start", url)
			default:
				cmd = exec.Command("xdg-open", url)
			}
			return cmd.Run()
		}
	}

	// fetch k8s version information
	{
		k8sVersion, err := c.k8s.GetServerVersion()
		if err != nil {
			return nil, fmt.Errorf("could not fetch k8s server version: %w", err)
		}
		c.tel.Attr("k8s_version", k8sVersion)
	}

	return c, nil
}

// Install handles the installation of Airbyte
func (c *Command) Install(ctx context.Context, user, pass string) error {
	if err := c.checkDocker(ctx); err != nil {
		return err
	}

	if err := c.handleChart(ctx, chartRequest{
		name:         "airbyte",
		repoName:     airbyteRepoName,
		repoURL:      airbyteRepoURL,
		chartName:    airbyteChartName,
		chartRelease: airbyteChartRelease,
		namespace:    airbyteNamespace,
	}); err != nil {
		return fmt.Errorf("could not install airbyte chart: %w", err)
	}

	if err := c.handleChart(ctx, chartRequest{
		name:         "nginx",
		repoName:     nginxRepoName,
		repoURL:      nginxRepoURL,
		chartName:    nginxChartName,
		chartRelease: nginxChartRelease,
		namespace:    nginxNamespace,
	}); err != nil {
		return fmt.Errorf("could not install nginx chart: %w", err)
	}

	spinnerIngress, err := pterm.DefaultSpinner.Start("ingress - installing")
	if err != nil {
		return fmt.Errorf("could not start ingress spinner: %w", err)
	}

	// basic auth
	if err := c.handleBasicAuthSecret(ctx, user, pass); err != nil {
		return fmt.Errorf("could not create or update basic-auth secret: %w", err)
	}

	if c.k8s.ExistsIngress(ctx, airbyteNamespace, airbyteIngress) {
		if err := c.k8s.UpdateIngress(ctx, airbyteNamespace, ingress()); err != nil {
			spinnerIngress.Fail("ingress - failed to update")
			return fmt.Errorf("could not update existing ingress: %w", err)
		}
		spinnerIngress.Success("ingress - updated")
	} else {
		if err := c.k8s.CreateIngress(ctx, airbyteNamespace, ingress()); err != nil {
			spinnerIngress.Fail("ingress - failed to install")
			return fmt.Errorf("could not create ingress: %w", err)
		}
		spinnerIngress.Success("ingress - installed")
	}

	return c.openBrowser(ctx, "http://localhost")
}

// handleBasicAuthSecret creates or updates the appropriate basic auth credentials for ingress.
func (c *Command) handleBasicAuthSecret(ctx context.Context, user, pass string) error {
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("could not hash basic auth password: %w", err)
	}

	data := map[string][]byte{"auth": []byte(fmt.Sprintf("%s:%s", user, hashedPass))}
	return c.k8s.CreateOrUpdateSecret(ctx, airbyteNamespace, "basic-auth", data)
}

// Uninstall handles the uninstallation of Airbyte.
func (c *Command) Uninstall(ctx context.Context) error {
	if err := c.checkDocker(ctx); err != nil {
		return err
	}

	{
		spinnerAb, err := pterm.DefaultSpinner.Start(fmt.Sprintf("helm - uninstalling airbyte chart %s", airbyteChartRelease))
		if err != nil {
			return fmt.Errorf("could not create spinner: %w", err)
		}

		airbyteChartExists := true
		if _, err := c.helm.GetRelease(airbyteChartRelease); err != nil {
			if !errors.Is(err, driver.ErrReleaseNotFound) {
				spinnerAb.Fail("helm - airbyte chart failed to fetch release")
				return fmt.Errorf("could not fetch airbyte release: %w", err)
			}
			airbyteChartExists = false
		}
		if airbyteChartExists {
			if err := c.helm.UninstallReleaseByName(airbyteChartRelease); err != nil {
				spinnerAb.Fail("helm - airbyte chart failed to uninstall")
				return fmt.Errorf("could not uninstall airbyte chart: %w", err)
			}
		}
		spinnerAb.Success()
	}

	{
		spinnerNginx, err := pterm.DefaultSpinner.Start(fmt.Sprintf("helm - uninstalling nginx chart %s", nginxChartRelease))
		if err != nil {
			return fmt.Errorf("coud not create spinner: %w", err)
		}

		nginxChartExists := true
		if _, err := c.helm.GetRelease(nginxChartRelease); err != nil {
			if !errors.Is(err, driver.ErrReleaseNotFound) {
				spinnerNginx.Fail("helm - nginx chart failed to fetch release")
				return fmt.Errorf("could not fetch nginx release: %w", err)
			}
			nginxChartExists = false
		}

		if nginxChartExists {
			if err := c.helm.UninstallReleaseByName(nginxChartRelease); err != nil {
				spinnerNginx.Fail("helm - nginx chart failed to uninstall")
				return fmt.Errorf("could not uninstall nginx chart: %w", err)
			}
		}
		spinnerNginx.Success()
	}

	spinnerNamespace, err := pterm.DefaultSpinner.Start(fmt.Sprintf("k8s - deleting namespace %s", airbyteNamespace))
	if err != nil {
		return fmt.Errorf("could not create spinner: %w", err)
	}

	if err := c.k8s.DeleteNamespace(ctx, airbyteNamespace); err != nil {
		if !k8serrors.IsNotFound(err) {
			spinnerNamespace.Fail()
			return fmt.Errorf("could not delete namespace: %w", err)
		}
	}

	// there is no blocking delete namespace call, so poll until it's been deleted or we've exhausted our time
	namespaceDeleted := false
	var wg sync.WaitGroup
	ticker := time.NewTicker(1 * time.Second) // how ofter to check
	timer := time.After(5 * time.Minute)      // how long to wait
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ticker.C:
				if !c.k8s.ExistsNamespace(ctx, airbyteNamespace) {
					namespaceDeleted = true
					return
				}
			case <-timer:
				ticker.Stop()
				return
			}
		}
	}()

	wg.Wait()

	if !namespaceDeleted {
		spinnerNamespace.Fail()
		return errors.New("could not delete namespace")
	}

	spinnerNamespace.Success()

	return nil
}

// checkDocker call the ServerVersion on the DockerClient.
// Will return ErrDocker if any error is caused by docker.
func (c *Command) checkDocker(ctx context.Context) error {
	spinner, err := pterm.DefaultSpinner.Start("docker - verifying")
	if err != nil {
		return fmt.Errorf("could not start spinner: %w", err)
	}

	ver, err := c.docker.ServerVersion(ctx)
	if err != nil {
		spinner.Fail("docker is not running")
		return errors.Join(ErrDocker, fmt.Errorf("docker is not running: %w", err))
	}

	c.tel.Attr("docker_version", ver.Version)
	c.tel.Attr("docker_arch", ver.Arch)
	c.tel.Attr("docker_platform", ver.Platform.Name)

	spinner.Success(fmt.Sprintf("docker - verified; version: %s", ver.Version))

	return nil
}

// chartRequest exists to make all the parameters to handleChart somewhat manageable
type chartRequest struct {
	name         string
	repoName     string
	repoURL      string
	chartName    string
	chartRelease string
	namespace    string
}

// handleChart will handle the installation of a chart
func (c *Command) handleChart(
	ctx context.Context,
	req chartRequest,
) error {
	spinner, err := pterm.DefaultSpinner.Start(fmt.Sprintf("helm - adding %s repository", req.name))
	if err != nil {
		return fmt.Errorf("could not start spinner: %w", err)
	}

	if err := c.helm.AddOrUpdateChartRepo(repo.Entry{
		Name: req.repoName,
		URL:  req.repoURL,
	}); err != nil {
		spinner.Fail(fmt.Sprintf("helm - could not add repo %s", req.repoName))
		return fmt.Errorf("could not add %s chart repo: %w", req.name, err)
	}

	spinner.UpdateText(fmt.Sprintf("helm - fetching chart %s", req.chartName))
	chart, _, err := c.helm.GetChart(req.chartName, &action.ChartPathOptions{})
	if err != nil {
		spinner.Fail(fmt.Sprintf("helm - could not fetch chart %s", req.chartName))
		return fmt.Errorf("could not fetch chart %s: %w", req.chartName, err)
	}

	c.tel.Attr(fmt.Sprintf("helm_%s_chart_version", req.name), chart.Metadata.Version)

	spinner.UpdateText(fmt.Sprintf("helm - installing chart %s (%s)", req.chartName, chart.Metadata.Version))
	release, err := c.helm.InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
		ReleaseName:     req.chartRelease,
		ChartName:       req.chartName,
		CreateNamespace: true,
		Namespace:       req.namespace,
		Wait:            true,
		Timeout:         10 * time.Minute,
	},
		&helmclient.GenericHelmOptions{},
	)
	if err != nil {
		spinner.Fail(fmt.Sprintf("helm - failed to install chart %s (%s)", req.chartName, chart.Metadata.Version))
		return fmt.Errorf("could not install helm: %w", err)
	}

	c.tel.Attr(fmt.Sprintf("helm_%s_release_version", req.name), strconv.Itoa(release.Version))

	spinner.Success(fmt.Sprintf("helm - chart installed; name: %s, namespace: %s, version: %d", release.Name, release.Namespace, release.Version))
	return nil
}

// openBrowser will open the url in the user's browser but only if the url returns a 200 response code first
// TODO: clean up this method, make it testable
func (c *Command) openBrowser(ctx context.Context, url string) error {
	spinner, err := pterm.DefaultSpinner.Start("browser - waiting for ingress")
	if err != nil {
		return fmt.Errorf("could not start browser spinner: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	alive := make(chan error)

	go func() {
		tick := time.Tick(1 * time.Second)
		for {
			select {
			case <-ctx.Done():
				alive <- fmt.Errorf("liveness check failed: %w", ctx.Err())
			case <-tick:
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				if err != nil {
					alive <- fmt.Errorf("could not create request: %w", err)
				}
				res, _ := c.http.Do(req)
				// if no auth, we should get a 200
				if res != nil && res.StatusCode == 200 {
					alive <- nil
				}
				// if basic auth, we should get a 401 with a specific header that contains abctl
				if res != nil && res.StatusCode == 401 && strings.Contains(res.Header.Get("WWW-Authenticate"), "abctl") {
					alive <- nil
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		spinner.Fail("browser - timed out")
		return fmt.Errorf("liveness check failed: %w", ctx.Err())
	case err := <-alive:
		if err != nil {
			spinner.Fail("browser - failed liveness check")
			return fmt.Errorf("failed liveness check: %w", err)
		}
	}
	// if we're here, then no errors occurred

	spinner.UpdateText("browser - launching")

	if err := c.launcher(url); err != nil {
		spinner.Fail(fmt.Sprintf("browser - failed to launch browser; please access %s directly", url))
		return fmt.Errorf("could not launch browser: %w", err)
	}

	spinner.Success("browser - launched")
	return nil
}

// ingress creates an ingress type for defining the webapp ingress rules.
func ingress() *networkingv1.Ingress {
	var pathType = networkingv1.PathType("Prefix")
	var ingressClassName = "nginx"

	return &networkingv1.Ingress{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      airbyteIngress,
			Namespace: airbyteNamespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/auth-type":   "basic",
				"nginx.ingress.kubernetes.io/auth-secret": "basic-auth",
				"nginx.ingress.kubernetes.io/auth-realm":  "Authentication Required - Airbyte (abctl)",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules: []networkingv1.IngressRule{
				{
					Host: "localhost",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: fmt.Sprintf("%s-airbyte-webapp-svc", airbyteChartRelease),
											Port: networkingv1.ServiceBackendPort{
												Name: "http",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// defaultDocker returns the default docker client
func defaultDocker(userHome string) (DockerClient, error) {
	var docker DockerClient
	var err error

	switch runtime.GOOS {
	case "darwin":
		// on mac, sometimes the docker host isn't set correctly, if it fails check the home directory
		docker, err = client.NewClientWithOpts(client.FromEnv, client.WithHost("unix:///var/run/docker.sock"))
		if err != nil {
			// keep the original error, as we'll join with the next error (if another error occurs)
			outerErr := err
			docker, err = client.NewClientWithOpts(client.FromEnv, client.WithHost(fmt.Sprintf("unix:///%s/.docker/run/docker.sock", userHome)))
			if err != nil {
				err = errors.Join(err, outerErr)
			}
		}
	case "windows":
		docker, err = client.NewClientWithOpts(client.FromEnv, client.WithHost("npipe:////./pipe/docker_engine"))
	default:
		docker, err = client.NewClientWithOpts(client.FromEnv, client.WithHost("unix:///var/run/docker.sock"))
	}
	if err != nil {
		return nil, errors.Join(ErrDocker, fmt.Errorf("could not create docker client: %w", err))
	}

	return docker, nil
}

// defaultK8s returns the default k8s client
func defaultK8s(userHome string) (k8s.K8sClient, error) {
	k8sCfg, err := k8sClientConfig(userHome)
	if err != nil {
		return nil, fmt.Errorf("could not create k8s client config: %w", err)
	}

	restCfg, err := k8sCfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not create k8s config client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("could not create k8s client: %w", err)
	}

	return &k8s.DefaultK8sClient{ClientSet: k8sClient}, nil
}

// defaultHelm returns the default helm client
func defaultHelm(userHome string) (HelmClient, error) {
	k8sCfg, err := k8sClientConfig(userHome)
	if err != nil {
		return nil, fmt.Errorf("could not create k8s client config: %w", err)
	}

	restCfg, err := k8sCfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not determine kubernetes client: %w", err)
	}

	helm, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options:    &helmclient.Options{Namespace: airbyteNamespace, Output: &noopWriter{}, DebugLog: func(format string, v ...interface{}) {}},
		RestConfig: restCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("coud not create helm client: %w", err)
	}

	return helm, nil
}

// k8sClientConfig returns a k8s client config using the ~/.kubc/config file and the k8sContext context.
func k8sClientConfig(userHome string) (clientcmd.ClientConfig, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: filepath.Join(userHome, ".kube", "config")},
		&clientcmd.ConfigOverrides{CurrentContext: k8sContext},
	), nil
}

// noopWriter is used by the helm client to suppress its verbose output
type noopWriter struct {
}

func (w *noopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
