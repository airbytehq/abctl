package local

import (
	"context"
	"errors"
	"fmt"
	k8s2 "github.com/airbytehq/abctl/internal/cmd/local/k8s"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/cli/browser"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/mittwald/go-helm-client/values"
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
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
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

// Port is the default port that Airbyte will deploy to.
const Port = 8000

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

// BrowserLauncher primarily for testing purposes.
type BrowserLauncher func(url string) error

// Command is the local command, responsible for installing, uninstalling, or other local actions.
type Command struct {
	provider         k8s2.Provider
	cluster          k8s2.Cluster
	http             HTTPClient
	helm             HelmClient
	k8s              k8s2.K8sClient
	portHTTP         int
	spinner          *pterm.SpinnerPrinter
	tel              telemetry.Client
	launcher         BrowserLauncher
	userHome         string
	helmChartVersion string
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
func WithK8sClient(client k8s2.K8sClient) Option {
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

func WithHelmChartVersion(version string) Option {
	return func(c *Command) {
		c.helmChartVersion = version
	}
}

func WithPortHTTP(port int) Option {
	return func(c *Command) {
		c.portHTTP = port
	}
}

// New creates a new Command
func New(provider k8s2.Provider, opts ...Option) (*Command, error) {
	c := &Command{provider: provider}
	for _, opt := range opts {
		opt(c)
	}

	// determine userhome if not defined
	if c.userHome == "" {
		var err error
		if c.userHome, err = os.UserHomeDir(); err != nil {
			return nil, fmt.Errorf("could not determine user home directory: %w", err)
		}
	}

	// set http client, if not defined
	if c.http == nil {
		c.http = &http.Client{Timeout: 10 * time.Second}
	}

	if c.portHTTP == 0 {
		c.portHTTP = Port
	}

	// set k8s client, if not defined
	if c.k8s == nil {
		kubecfg := filepath.Join(c.userHome, provider.Kubeconfig)
		var err error
		if c.k8s, err = defaultK8s(kubecfg, provider.Context); err != nil {
			return nil, err
		}
	}

	// set the helm client, if not defined
	if c.helm == nil {
		kubecfg := filepath.Join(c.userHome, provider.Kubeconfig)
		var err error
		if c.helm, err = defaultHelm(kubecfg, provider.Context); err != nil {
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

	if c.helmChartVersion == "latest" {
		c.helmChartVersion = ""
	}

	// fetch k8s version information
	{
		k8sVersion, err := c.k8s.GetServerVersion()
		if err != nil {
			return nil, fmt.Errorf("%w: could not fetch kubernetes server version: %w", localerr.ErrKubernetes, err)
		}
		c.tel.Attr("k8s_version", k8sVersion)
	}

	// set provider version
	c.tel.Attr("provider", provider.Name)

	return c, nil
}

// Install handles the installation of Airbyte
func (c *Command) Install(ctx context.Context, user, pass string) error {
	if err := c.handleChart(ctx, chartRequest{
		name:         "airbyte",
		repoName:     airbyteRepoName,
		repoURL:      airbyteRepoURL,
		chartName:    airbyteChartName,
		chartRelease: airbyteChartRelease,
		chartVersion: c.helmChartVersion,
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
		values:       append(c.provider.HelmNginx, fmt.Sprintf("controller.service.ports.http=%d", c.portHTTP)),
	}); err != nil {
		// If we timed out, there is a good chance it's due to an unavailable port, check if this is the case.
		// As the kubernetes client doesn't return usable error types, have to check for a specific string value.
		if strings.Contains(err.Error(), "client rate limiter Wait returned an error") {
			pterm.Warning.Printfln("Encountered an error while installing the %s Helm Chart.\n"+
				"This could be an indication that port %d is not available.\n"+
				"If installation fails, please try again with a different port.", nginxChartName, c.portHTTP)

			srv, err := c.k8s.GetService(ctx, nginxNamespace, "ingress-nginx-controller")
			// If there is an error, we can ignore it as we only are checking for a missing ingress entry,
			// and an error would indicate the inability to check for that entry.
			if err == nil {
				ingresses := srv.Status.LoadBalancer.Ingress
				if len(ingresses) == 0 {
					// if there are no ingresses, that is a possible indicator that the port is already in use.
					return fmt.Errorf("%w: could not install nginx chart", localerr.ErrIngress)
				}
			}
		}
		return fmt.Errorf("could not install nginx chart: %w", err)
	}

	c.spinner.UpdateText("Configuring Basic-Auth")
	// basic auth
	if err := c.handleBasicAuthSecret(ctx, user, pass); err != nil {
		return fmt.Errorf("could not create or update basic-auth secret: %w", err)
	}

	c.spinner.UpdateText("Checking for existing Ingress")

	if c.k8s.ExistsIngress(ctx, airbyteNamespace, airbyteIngress) {
		pterm.Success.Println("Found existing Ingress")
		if err := c.k8s.UpdateIngress(ctx, airbyteNamespace, ingress()); err != nil {
			pterm.Error.Printfln("Unable to update existing Ingress")
			return fmt.Errorf("could not update existing ingress: %w", err)
		}
		pterm.Success.Println("Updated existing Ingress")
	} else {
		pterm.Info.Println("No existing Ingress found, will create one")
		if err := c.k8s.CreateIngress(ctx, airbyteNamespace, ingress()); err != nil {
			pterm.Error.Println("Unable to create ingress")
			return fmt.Errorf("could not create ingress: %w", err)
		}
		pterm.Success.Println("Ingress created")
	}

	c.spinner.UpdateText("Verifying ingress")
	if err := c.openBrowser(ctx, fmt.Sprintf("http://localhost:%d", c.portHTTP)); err != nil {
		return err
	}

	return nil
}

// handleBasicAuthSecret creates or updates the appropriate basic auth credentials for ingress.
func (c *Command) handleBasicAuthSecret(ctx context.Context, user, pass string) error {
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		pterm.Error.Println("Basic Auth secret could not be hashed.\n" +
			"This may indicate an issue with the username or password provided.\n" +
			"Please provider different credentials and try again.")

		return fmt.Errorf("could not hash basic auth password: %w", err)
	}

	data := map[string][]byte{"auth": []byte(fmt.Sprintf("%s:%s", user, hashedPass))}
	if err := c.k8s.CreateOrUpdateSecret(ctx, airbyteNamespace, "basic-auth", data); err != nil {
		pterm.Error.Println("Could not create Basic-Auth secret")
	}
	pterm.Success.Println("Basic-Auth secret created")
	return nil
}

// Uninstall handles the uninstallation of Airbyte.
func (c *Command) Uninstall(ctx context.Context) error {
	{
		c.spinner.UpdateText(fmt.Sprintf("Verifying %s Helm Chart installation status", airbyteChartName))

		airbyteChartExists := true
		if _, err := c.helm.GetRelease(airbyteChartRelease); err != nil {
			if !errors.Is(err, driver.ErrReleaseNotFound) {
				pterm.Error.Printfln("Could not verify installation status of %s Helm Chart", airbyteChartName)
				return fmt.Errorf("could not fetch airbyte release: %w", err)
			}

			pterm.Success.Printfln("Helm Chart %s is not installed", airbyteChartName)
			airbyteChartExists = false
		} else {
			pterm.Success.Printfln("Verified Helm Chart %s is installed", airbyteChartName)
		}

		if airbyteChartExists {
			c.spinner.UpdateText(fmt.Sprintf("Uninstalling %s Helm Chart", airbyteChartName))
			if err := c.helm.UninstallReleaseByName(airbyteChartRelease); err != nil {
				pterm.Error.Printfln("Could not uninstall %s Helm Chart", airbyteChartName)
				return fmt.Errorf("could not uninstall airbyte chart: %w", err)
			}
			pterm.Success.Printfln("Uninstalled %s Helm Chart", airbyteChartName)
		}
	}

	{
		c.spinner.UpdateText(fmt.Sprintf("Verifying %s Helm Chart installation status", nginxChartName))

		nginxChartExists := true
		if _, err := c.helm.GetRelease(nginxChartRelease); err != nil {
			if !errors.Is(err, driver.ErrReleaseNotFound) {
				pterm.Error.Printfln("Could not verify installation status of %s Helm Chart", nginxChartName)
				return fmt.Errorf("could not fetch nginx release: %w", err)
			}

			pterm.Success.Printfln("Helm Chart %s is not installed", nginxChartName)
			nginxChartExists = false
		}

		if nginxChartExists {
			c.spinner.UpdateText(fmt.Sprintf("Uninstalling %s Helm Chart", nginxChartName))
			if err := c.helm.UninstallReleaseByName(nginxChartRelease); err != nil {
				pterm.Error.Printfln("Could not uninstall %s Helm Chart", nginxChartName)
				return fmt.Errorf("could not uninstall nginx chart: %w", err)
			}
		}
		pterm.Success.Printfln("Uninstalled %s Helm Chart", nginxChartName)
	}

	c.spinner.UpdateText(fmt.Sprintf("Deleting Kubernetes namespace '%s'", airbyteNamespace))

	if err := c.k8s.DeleteNamespace(ctx, airbyteNamespace); err != nil {
		if !k8serrors.IsNotFound(err) {
			pterm.Error.Printfln("Could not delete Kubernetes namespace '%s'", airbyteNamespace)
			return fmt.Errorf("could not delete namespace: %w", err)
		}
	}

	// there is no blocking delete namespace call, so poll until it's been deleted, or we've exhausted our time
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
		pterm.Error.Printfln("Could not delete Kubernetes namespace '%s'", airbyteNamespace)
		return errors.New("could not delete namespace")
	}

	pterm.Success.Printfln("Namespace '%s' deleted", airbyteNamespace)

	return nil
}

// chartRequest exists to make all the parameters to handleChart somewhat manageable
type chartRequest struct {
	name         string
	repoName     string
	repoURL      string
	chartName    string
	chartRelease string
	chartVersion string
	namespace    string
	values       []string
}

// handleChart will handle the installation of a chart
func (c *Command) handleChart(
	ctx context.Context,
	req chartRequest,
) error {
	c.spinner.UpdateText(fmt.Sprintf("Configuring %s Helm repository", req.name))

	if err := c.helm.AddOrUpdateChartRepo(repo.Entry{
		Name: req.repoName,
		URL:  req.repoURL,
	}); err != nil {
		pterm.Error.Printfln("Unable to configure %s Helm repository", req.repoName)
		return fmt.Errorf("could not add %s chart repo: %w", req.name, err)
	}

	c.spinner.UpdateText(fmt.Sprintf("Fetching %s Helm Chart", req.chartName))
	helmChart, _, err := c.helm.GetChart(req.chartName, &action.ChartPathOptions{Version: req.chartVersion})
	if err != nil {
		pterm.Error.Printfln("Unable to fetch %s Helm Chart", req.chartName)
		return fmt.Errorf("could not fetch chart %s: %w", req.chartName, err)
	}

	c.tel.Attr(fmt.Sprintf("helm_%s_chart_version", req.name), helmChart.Metadata.Version)

	c.spinner.UpdateText(fmt.Sprintf("Installing '%s' (version: %s) Helm Chart", req.chartName, helmChart.Metadata.Version))
	helmRelease, err := c.helm.InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
		ReleaseName:     req.chartRelease,
		ChartName:       req.chartName,
		CreateNamespace: true,
		Namespace:       req.namespace,
		Wait:            true,
		Timeout:         10 * time.Minute,
		ValuesOptions:   values.Options{Values: req.values},
		Version:         req.chartVersion,
	},
		&helmclient.GenericHelmOptions{},
	)
	if err != nil {
		pterm.Error.Printfln("Failed to install %s Helm Chart", req.chartName)
		return fmt.Errorf("could not install helm: %w", err)
	}

	c.tel.Attr(fmt.Sprintf("helm_%s_release_version", req.name), strconv.Itoa(helmRelease.Version))

	pterm.Success.Printfln(
		"Installed Helm Chart %s:\n\tname: %s\n\tnamespace: %s\n\tversion: %s\n\trelease: %d",
		req.chartName, helmRelease.Name, helmRelease.Namespace, helmRelease.Chart.Metadata.Version, helmRelease.Version)
	return nil
}

// openBrowser will open the url in the user's browser but only if the url returns a 200 response code first
// TODO: clean up this method, make it testable
func (c *Command) openBrowser(ctx context.Context, url string) error {
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
				if res != nil && res.StatusCode == http.StatusOK {
					alive <- nil
				}
				// if basic auth, we should get a 401 with a specific header that contains abctl
				if res != nil && res.StatusCode == http.StatusUnauthorized && strings.Contains(res.Header.Get("WWW-Authenticate"), "abctl") {
					alive <- nil
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		pterm.Error.Println("Timed out waiting for ingress")
		return fmt.Errorf("browser liveness check failed: %w", ctx.Err())
	case err := <-alive:
		if err != nil {
			pterm.Error.Println("Ingress verification failed")
			return fmt.Errorf("browser failed liveness check: %w", err)
		}
	}
	// if we're here, then no errors occurred

	c.spinner.UpdateText(fmt.Sprintf("Attempting to launch web-browser for %s", url))

	if err := c.launcher(url); err != nil {
		pterm.Warning.Printfln("Failed to launch web-browser.\n"+
			"Please launch your web-browser to access %s", url)
		pterm.Debug.Printfln("failed to launch web-browser: %s", err.Error())
		// don't consider a failed web-browser to be a failed installation
	}

	pterm.Success.Println("Launched web-browser successfully")

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

// defaultK8s returns the default k8s client
func defaultK8s(kubecfg, kubectx string) (k8s2.K8sClient, error) {
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

	return &k8s2.DefaultK8sClient{ClientSet: k8sClient}, nil
}

// defaultHelm returns the default helm client
func defaultHelm(kubecfg, kubectx string) (HelmClient, error) {
	k8sCfg, err := k8sClientConfig(kubecfg, kubectx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", localerr.ErrKubernetes, err)
	}

	restCfg, err := k8sCfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("%w: could not create rest config: %w", localerr.ErrKubernetes, err)
	}

	helm, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options:    &helmclient.Options{Namespace: airbyteNamespace, Output: &noopWriter{}, DebugLog: func(format string, v ...interface{}) {}},
		RestConfig: restCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create helm client: %w", err)
	}

	return helm, nil
}

// k8sClientConfig returns a k8s client config using the ~/.kubc/config file and the k8sContext context.
func k8sClientConfig(kubecfg, kubectx string) (clientcmd.ClientConfig, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubecfg},
		&clientcmd.ConfigOverrides{CurrentContext: kubectx},
	), nil
}

// noopWriter is used by the helm client to suppress its verbose output
type noopWriter struct {
}

func (w *noopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
