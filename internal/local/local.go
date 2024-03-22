package local

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
	"runtime"
	"time"
)

const (
	airbyteChartName    = "airbyte/airbyte"
	airbyteChartRelease = "airbyte-abctl"
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
	helm helmclient.Client
	k8s  *kubernetes.Clientset
}

func New() (*Command, error) {
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
		return nil, errors.Wrap(err, "could not create k8s client")
	}

	helm, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options:    &helmclient.Options{Namespace: airbyteNamespace, Output: &nilWriter{}, DebugLog: func(format string, v ...interface{}) {}},
		RestConfig: restCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("coud not create helm client: %w", err)
	}
	return &Command{helm: helm, k8s: k8s}, nil
}

func (lc *Command) Install() error {
	if err := checkDocker(); err != nil {
		return err
	}

	if err := lc.handleChart("airbyte", airbyteRepoName, airbyteRepoURL, airbyteChartName, airbyteChartRelease, airbyteNamespace); err != nil {
		return fmt.Errorf("could not install airbyte chart: %w", err)
	}

	if err := lc.handleChart("nginx", nginxRepoName, nginxRepoURL, nginxChartName, nginxChartRelease, nginxNamespace); err != nil {
		return fmt.Errorf("could not install nginx chart: %w", err)
	}

	spinnerIngress, err := pterm.DefaultSpinner.Start("ingress - installing")
	if err != nil {
		return fmt.Errorf("could not start ingress spinner: %w", err)
	}
	if _, err := lc.k8s.NetworkingV1().Ingresses(airbyteNamespace).Create(context.Background(), ingress(), v1.CreateOptions{}); err != nil {
		spinnerIngress.Fail("ingress - failed to install")
		return fmt.Errorf("could not create ingress: %w", err)
	}
	spinnerIngress.Success("ingress - installed")

	return openBrowser("http://localhost")
}

func (lc *Command) Uninstall() error {
	spinnerAb, err := pterm.DefaultSpinner.Start(fmt.Sprintf("helm - uninstalling airbyte chart %s", airbyteChartRelease))
	if err != nil {
		return fmt.Errorf("could not create spinner: %w", err)
	}
	if err := lc.helm.UninstallReleaseByName(airbyteChartRelease); err != nil {
		spinnerAb.Fail("helm - airbyte chart failed to uninstall")
		return fmt.Errorf("could not uninstall airbyte chart: %w", err)
	}
	spinnerAb.Success()

	spinnerNginx, err := pterm.DefaultSpinner.Start(fmt.Sprintf("helm - uninstalling nginx chart %s", nginxChartRelease))
	if err := lc.helm.UninstallReleaseByName(nginxChartRelease); err != nil {
		spinnerAb.Fail("helm - nginx chart failed to uninstall")
		return fmt.Errorf("could not uninstall nginx chart: %w", err)
	}
	spinnerNginx.Success()

	spinnerNamespace, err := pterm.DefaultSpinner.Start(fmt.Sprintf("k8s - deleting namespace %s", airbyteNamespace))
	if err != nil {
		return fmt.Errorf("could not create spinner: %w", err)
	}
	if err := lc.k8s.CoreV1().Namespaces().Delete(context.Background(), airbyteNamespace, v1.DeleteOptions{}); err != nil {
		spinnerNamespace.Fail()
		return fmt.Errorf("could not delete namespace: %w", err)
	}
	// there is no blocking delete namespace call, so lets do this the old-fashioned way
	for {
		_, err = lc.k8s.CoreV1().Namespaces().Get(context.Background(), airbyteNamespace, v1.GetOptions{})
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

func checkDocker() error {
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

	ping, err := docker.Ping(context.Background())
	if err != nil {
		spinner.Fail("docker is not running")
		return fmt.Errorf("docker is not running: %w", err)
	}

	spinner.Success(fmt.Sprintf("docker - verified; api version: %s", ping.APIVersion))

	return nil
}

func (lc *Command) handleChart(
	name string,
	repoName string,
	repoURL string,
	chartName string,
	chartRelease string,
	namespace string,
) error {
	spinner, err := pterm.DefaultSpinner.Start(fmt.Sprintf("helm - adding %s repository", name))
	if err != nil {
		return fmt.Errorf("could not start spinner: %w", err)
	}

	if err := lc.helm.AddOrUpdateChartRepo(repo.Entry{
		Name: repoName,
		URL:  repoURL,
	}); err != nil {
		spinner.Fail(fmt.Sprintf("helm - could not add repo %s", repoName))
		return fmt.Errorf("could not add %s chart repo: %w", name, err)
	}

	spinner.UpdateText(fmt.Sprintf("helm - fetching chart %s", chartName))
	chart, _, err := lc.helm.GetChart(chartName, &action.ChartPathOptions{})
	if err != nil {
		spinner.Fail(fmt.Sprintf("helm - could not fetch chart %s", chartName))
		return fmt.Errorf("could not fetch chart %s: %w", chartName, err)
	}

	spinner.UpdateText(fmt.Sprintf("helm - installing chart %s (%s)", chartName, chart.Metadata.Version))
	release, err := lc.helm.InstallOrUpgradeChart(context.Background(), &helmclient.ChartSpec{
		ReleaseName:     chartRelease,
		ChartName:       chartName,
		CreateNamespace: true,
		Namespace:       namespace,
		Wait:            true,
		Timeout:         10 * time.Minute,
	},
		&helmclient.GenericHelmOptions{},
	)
	if err != nil {
		spinner.Fail(fmt.Sprintf("helm - failed to install chart %s (%s)", chartName, chart.Metadata.Version))
		return errors.Wrap(err, "could not install helm")
	}
	spinner.Success(fmt.Sprintf("helm - chart installed; name: %s, namespace: %s, version: %d", release.Name, release.Namespace, release.Version))

	return nil
}

func ingress() *networkingv1.Ingress {
	var pathType = networkingv1.PathType("Prefix")
	var ingressClassName = "nginx"

	return &networkingv1.Ingress{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      "ingress-airbyte-webapp",
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

func openBrowser(url string) error {
	pterm.Println("opening browser")
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
