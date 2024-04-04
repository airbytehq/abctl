package local

import (
	"airbyte.io/abctl/internal/telemetry"
	"context"
	"fmt"
	"github.com/docker/docker/client"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
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

// nilWriter is used by the helm client to suppress its verbose output
type nilWriter struct {
}

func (w *nilWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

type Command struct {
	tel  telemetry.Client
	h    http.Client
	helm helmclient.Client
	k8s  *kubernetes.Clientset
}

func New(tel telemetry.Client) (*Command, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf(" could not determine user home directory: %w", err)
	}

	k8sCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: fmt.Sprintf("%s/.kube/config", userHome),
		},
		&clientcmd.ConfigOverrides{
			CurrentContext: k8sContext,
		},
	)

	var restCfg *rest.Config
	restCfg, err = k8sCfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not determine kubernetes client: %w", err)
	}
	k8s, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("could not create k8s client: %w", err)
	}

	// fetch k8s version information
	{
		discClient, err := discovery.NewDiscoveryClientForConfig(restCfg)
		if err != nil {
			return nil, fmt.Errorf("could not create k8s discovery client: %w", err)
		}
		k8sVersion, err := discClient.ServerVersion()
		if err != nil {
			return nil, fmt.Errorf("could not fetch k8s server version: %w", err)
		}
		tel.Attr("k8s_version", k8sVersion.String())
	}

	helm, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options:    &helmclient.Options{Namespace: airbyteNamespace, Output: &nilWriter{}, DebugLog: func(format string, v ...interface{}) {}},
		RestConfig: restCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("coud not create helm client: %w", err)
	}

	return &Command{
		tel:  tel,
		h:    http.Client{Timeout: 10 * time.Second},
		helm: helm,
		k8s:  k8s,
	}, nil
}

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

	_, err = c.k8s.NetworkingV1().Ingresses(airbyteNamespace).Get(ctx, airbyteIngress, v1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		// the ingress does not exist, create it
		if _, err := c.k8s.NetworkingV1().Ingresses(airbyteNamespace).Create(ctx, ingress(), v1.CreateOptions{}); err != nil {
			spinnerIngress.Fail("ingress - failed to install")
			return fmt.Errorf("could not create ingress: %w", err)
		}
		spinnerIngress.Success("ingress - installed")
	} else if err != nil {
		// some other error happened, return
		spinnerIngress.Fail("ingress - unable to fetch existing")
		return fmt.Errorf("could not fetch potential existing ingress: %w", err)
	} else {
		// ingress already exists, update it
		if _, err := c.k8s.NetworkingV1().Ingresses(airbyteNamespace).Update(ctx, ingress(), v1.UpdateOptions{}); err != nil {
			spinnerIngress.Fail("ingress - failed to update")
			return fmt.Errorf("could not update existing ingress: %w", err)
		}
		spinnerIngress.Success("ingress - updated")
	}

	return c.openBrowser(ctx, "http://localhost")
}

// TODO: add helm version data to telemetry
func (c *Command) Uninstall(ctx context.Context) error {
	if err := c.checkDocker(ctx); err != nil {
		return err
	}

	spinnerAb, err := pterm.DefaultSpinner.Start(fmt.Sprintf("helm - uninstalling airbyte chart %s", airbyteChartRelease))
	if err != nil {
		return fmt.Errorf("could not create spinner: %w", err)
	}
	if err := c.helm.UninstallReleaseByName(airbyteChartRelease); err != nil {
		spinnerAb.Fail("helm - airbyte chart failed to uninstall")
		return fmt.Errorf("could not uninstall airbyte chart: %w", err)
	}
	spinnerAb.Success()

	spinnerNginx, err := pterm.DefaultSpinner.Start(fmt.Sprintf("helm - uninstalling nginx chart %s", nginxChartRelease))
	if err := c.helm.UninstallReleaseByName(nginxChartRelease); err != nil {
		spinnerAb.Fail("helm - nginx chart failed to uninstall")
		return fmt.Errorf("could not uninstall nginx chart: %w", err)
	}
	spinnerNginx.Success()

	spinnerNamespace, err := pterm.DefaultSpinner.Start(fmt.Sprintf("k8s - deleting namespace %s", airbyteNamespace))
	if err != nil {
		return fmt.Errorf("could not create spinner: %w", err)
	}

	if err := c.k8s.CoreV1().Namespaces().Delete(ctx, airbyteNamespace, v1.DeleteOptions{}); err != nil {
		spinnerNamespace.Fail()
		return fmt.Errorf("could not delete namespace: %w", err)
	}

	// there is no blocking delete namespace call, so lets do this the old-fashioned way
	for {
		_, err = c.k8s.CoreV1().Namespaces().Get(ctx, airbyteNamespace, v1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				break
			} else {
				spinnerNamespace.Fail()
				return fmt.Errorf("error fetching namespace: %w", err)
			}
		} else {
			// old-fashioned!
			time.Sleep(1 * time.Second)
		}
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
	userHome, err := os.UserHomeDir()
	if err != nil {
		spinner.Fail("docker verification failed, could not determine home directory")
		return fmt.Errorf("could not determine user home directory: %w", err)
	}
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(fmt.Sprintf("unix://%s/.docker/run/docker.sock", userHome)))
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
