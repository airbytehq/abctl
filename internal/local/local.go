package local

import (
	"airbyte.io/abctl/internal/telemetry"
	"context"
	"fmt"
	"github.com/docker/docker/client"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
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
	"runtime"
	"strconv"
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

// Command is the local command, responsible for installing, uninstalling, or other local actions.
type Command struct {
	h        *http.Client
	helm     HelmClient
	k8s      K8sClient
	tel      telemetry.Client
	userHome string
}

type Option func(*Command)

// WithTelemetryClient define the telemetry client for this command.
func WithTelemetryClient(client telemetry.Client) Option {
	return func(c *Command) {
		c.tel = client
	}
}

// WithHTTPClient define the http client for this command.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Command) {
		c.h = client
	}
}

// WithHelmClient define the helm client for this command.
func WithHelmClient(client HelmClient) Option {
	return func(c *Command) {
		c.helm = client
	}
}

// WithK8sClient define the k8s client for this command.
func WithK8sClient(client K8sClient) Option {
	return func(c *Command) {
		c.k8s = client
	}
}

func New(opts ...Option) (*Command, error) {
	c := &Command{}
	for _, opt := range opts {
		opt(c)
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine user home directory: %w", err)
	}
	c.userHome = userHome

	// set http client, if not defined
	if c.h == nil {
		c.h = &http.Client{Timeout: 10 * time.Second}
	}

	// set k8s client, if not defined
	if c.k8s == nil {
		k8sCfg, err := k8sClientConfig(c.userHome)
		if err != nil {
			return nil, fmt.Errorf("could not create k8s client config: %w", err)
		}

		restCfg, err := k8sCfg.ClientConfig()
		k8s, err := kubernetes.NewForConfig(restCfg)
		if err != nil {
			return nil, fmt.Errorf("could not create k8s client: %w", err)
		}

		c.k8s = &defaultK8sClient{k8s: k8s}
	}

	// set the helm client, if not defined
	if c.helm == nil {
		k8sCfg, err := k8sClientConfig(c.userHome)
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

		c.helm = helm
	}

	// set telemetry client, if not defined
	if c.tel == nil {
		c.tel = telemetry.NoopClient{}
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

// k8sClientConfig returns a k8s client config using the ~/.kubc/config file and the k8sContext context.
func k8sClientConfig(userHome string) (clientcmd.ClientConfig, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: fmt.Sprintf("%s/.kube/config", userHome)},
		&clientcmd.ConfigOverrides{CurrentContext: k8sContext},
	), nil
}

// Install handles the installation of Airbyte
func (c *Command) Install(ctx context.Context) error {
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

func (c *Command) checkDocker(ctx context.Context) error {
	spinner, err := pterm.DefaultSpinner.Start("docker - verifying")
	if err != nil {
		return fmt.Errorf("could not start spinner: %w", err)
	}

	// TODO: remove this hack, docker-desktop on mac doesn't always correctly create the /var/run/docker.sock path,
	// so instead search for the ~/.docker/run/docker.sock
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(fmt.Sprintf("unix://%s/.docker/run/docker.sock", c.userHome)))
	if err != nil {
		spinner.Fail("docker verification failed, cold not create docker client")
		return fmt.Errorf("could not create docker client: %w", err)
	}

	ver, err := docker.ServerVersion(ctx)
	if err != nil {
		spinner.Fail("docker is not running")
		return fmt.Errorf("docker is not running: %w", err)
	}

	c.tel.Attr("docker_version", ver.Version)
	c.tel.Attr("docker_arch", ver.Arch)
	c.tel.Attr("docker_platform", ver.Platform.Name)

	spinner.Success(fmt.Sprintf("docker - verified; version: %s", ver.Version))

	return nil
}

type chartRequest struct {
	name         string
	repoName     string
	repoURL      string
	chartName    string
	chartRelease string
	namespace    string
}

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

// ingress creates an ingress type for defining the webapp ingress rules.
func ingress() *networkingv1.Ingress {
	var pathType = networkingv1.PathType("Prefix")
	var ingressClassName = "nginx"

	return &networkingv1.Ingress{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      airbyteIngress,
			Namespace: airbyteNamespace,
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
				res, _ := c.h.Do(req)
				if res != nil && res.StatusCode == 200 {
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

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Run(); err != nil {
		spinner.Fail("browser - failed to launch browser; please access http://localhost directly")
		return fmt.Errorf("could not launch browser: %w", err)
	}

	spinner.Success("browser - launched")
	return nil
}

// noopWriter is used by the helm client to suppress its verbose output
type noopWriter struct {
}

func (w *noopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
