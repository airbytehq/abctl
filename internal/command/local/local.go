package local

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"
	networkingv1 "k8s.io/api/networking/v1"
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
	//helm, err := helmclient.New(&helmclient.Options{Namespace: namespace})
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Wrap(err, "could not determine user home directory")
	}
	//k8sConfig, err := os.ReadFile(fmt.Sprintf("%s/.kube/config", userHome))
	//if err != nil {
	//	return nil, errors.Wrap(err, "could not read ~/.kube/config")
	//}

	kconf := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: fmt.Sprintf("%s/.kube/config", userHome),
		},
		&clientcmd.ConfigOverrides{
			CurrentContext: k8sContext,
		},
	)

	var restCfg *rest.Config
	restCfg, err = kconf.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not determine kubernetes client")
	}
	k8s, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not create k8s client")
	}

	helm, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options:    &helmclient.Options{Namespace: airbyteNamespace, Output: &nilWriter{}, DebugLog: func(format string, v ...interface{}) {}},
		RestConfig: restCfg,
	})
	//helm, err := helmclient.NewClientFromKubeConf(&helmclient.KubeConfClientOptions{
	//	KubeContext: k8sContext,
	//	KubeConfig:  k8sConfig,
	//	Options: &helmclient.Options{
	//		Namespace: k8sNamespace,
	//	},
	//})
	if err != nil {
		return nil, errors.Wrap(err, "could not create helm client")
	}
	return &Command{helm: helm, k8s: k8s}, nil
}

func (lc *Command) Install() error {
	if err := checkDocker(); err != nil {
		return err
	}

	if err := lc.helm.AddOrUpdateChartRepo(repo.Entry{
		Name: airbyteRepoName,
		URL:  airbyteRepoURL,
	}); err != nil {
		return errors.Wrap(err, "could not add airbyte chart repo")
	}

	if err := lc.helm.AddOrUpdateChartRepo(repo.Entry{
		Name: nginxRepoName,
		URL:  nginxRepoURL,
	}); err != nil {
		return errors.Wrap(err, "could not add nginx chart repo")
	}

	fmt.Printf("fetching chart %s... ", airbyteChartName)
	airbyteChart, _, err := lc.helm.GetChart(airbyteChartName, &action.ChartPathOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not fetch chart %s", airbyteChartName))
	}
	fmt.Printf("successfully fetched chart %s\n", airbyteChartName)
	fmt.Printf(" version: %s\n", airbyteChart.Metadata.Version)
	fmt.Printf(" app-version: %s\n", airbyteChart.Metadata.AppVersion)

	fmt.Printf("fetching chart %s... ", nginxChartName)
	nginxChart, _, err := lc.helm.GetChart(nginxChartName, &action.ChartPathOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not fetch chart %s", nginxChartName))
	}
	fmt.Printf("successfully fetched chart %s\n", nginxChartName)
	fmt.Printf(" version: %s\n", nginxChart.Metadata.Version)
	fmt.Printf(" app-version: %s\n", nginxChart.Metadata.AppVersion)

	//fmt.Printf("starting k3d cluster... ")
	//if err := k3d.ClusterRun(context.Background(), runtimes.SelectedRuntime, cluster); err != nil {
	//	return errors.Wrap(err, fmt.Sprintf("could not create k3d cluster %s", clusterName))
	//}
	//if err != nil {
	//	return errors.Wrap(err, "could not communicate with k3d")
	//}
	//fmt.Println("k3d cluster successfully created")

	fmt.Printf("installing chart %s (%s)... ", airbyteChartName, airbyteChart.Metadata.Version)
	airbyteRelease, err := lc.helm.InstallOrUpgradeChart(context.Background(), &helmclient.ChartSpec{
		ReleaseName:     airbyteChartRelease,
		ChartName:       airbyteChartName,
		CreateNamespace: true,
		Namespace:       airbyteNamespace,
		Wait:            true,
		Timeout:         10 * time.Minute,
	},
		&helmclient.GenericHelmOptions{},
	)
	if err != nil {
		return errors.Wrap(err, "could not install helm")
	}
	fmt.Printf("successfully installed chart %s\n", airbyteChartName)
	fmt.Printf(" name: %s\n", airbyteRelease.Name)
	fmt.Printf(" namespace: %s\n", airbyteRelease.Namespace)
	fmt.Printf(" version: %d\n", airbyteRelease.Version)

	fmt.Printf("installing chart %s (%s)... ", nginxChartName, nginxChart.Metadata.Version)
	nginxRelease, err := lc.helm.InstallOrUpgradeChart(context.Background(), &helmclient.ChartSpec{
		ReleaseName:     nginxChartRelease,
		ChartName:       nginxChartName,
		CreateNamespace: true,
		Namespace:       nginxNamespace,
		Wait:            true,
		Timeout:         10 * time.Minute,
	},
		&helmclient.GenericHelmOptions{},
	)
	if err != nil {
		return errors.Wrap(err, "could not install helm")
	}
	fmt.Printf("successfully installed chart %s\n", nginxChartName)
	fmt.Printf(" name: %s\n", nginxRelease.Name)
	fmt.Printf(" namespace: %s\n", nginxRelease.Namespace)
	fmt.Printf(" version: %d\n", nginxRelease.Version)

	fmt.Printf("creating ingress... ")
	lc.k8s.NetworkingV1().Ingresses(airbyteNamespace).Create(context.Background(), ingress(), v1.CreateOptions{})
	fmt.Println("success")

	return openBrowser("http://localhost")

	//return nil
}

func (lc *Command) Uninstall() error {
	fmt.Printf("uninstalling helm %s... ", airbyteChartRelease)
	if err := lc.helm.UninstallReleaseByName(airbyteChartRelease); err != nil {
		return errors.Wrap(err, "could not uninstall airbyte helm")
	}
	fmt.Println("success")

	fmt.Printf("uninstalling helm %s... ", nginxChartRelease)
	if err := lc.helm.UninstallReleaseByName(nginxChartRelease); err != nil {
		return errors.Wrap(err, "could not uninstall nginx helm")
	}
	fmt.Println("success")

	fmt.Printf("deleting namespace %s... ", airbyteNamespace)
	if err := lc.k8s.CoreV1().Namespaces().Delete(context.Background(), airbyteNamespace, v1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "could not delete namespace")
	}
	//if err := k3d.ClusterDelete(context.Background(), runtimes.SelectedRuntime, &cluster.Cluster, types.ClusterDeleteOpts{
	//	SkipRegistryCheck: false,
	//}); err != nil {
	//	return errors.Wrap(err, fmt.Sprintf("could not delete cluster %s", clusterName))
	//}
	fmt.Println("success")
	return nil
}

func checkDocker() error {
	fmt.Println("checking docker...")
	// TODO: remove this hack, docker-desktop on mac doesn't always correctly create the /var/run/docker.sock path, so
	// instead search for the ~/.docker/run/docker.sock
	userHome, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "could not determine user home directory")
	}
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(fmt.Sprintf("unix://%s/.docker/run/docker.sock", userHome)))
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	ping, err := docker.Ping(context.Background())
	if err != nil {
		return errors.Wrap(err, "docker is not running")
	}

	fmt.Println("docker found and running")
	fmt.Printf(" api version: %s\n", ping.APIVersion)

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

//var (
//	cluster, _ = config.TransformSimpleToClusterConfig(
//		context.Background(),
//		runtimes.SelectedRuntime,
//		v1alpha5.SimpleConfig{
//			TypeMeta: k3dTypes.TypeMeta{
//				Kind:       "Simple",
//				APIVersion: config.DefaultConfigApiVersion,
//			},
//			ObjectMeta: k3dTypes.ObjectMeta{
//				Name: clusterName,
//			},
//			ExposeAPI: v1alpha5.SimpleExposureOpts{},
//			Ports: []v1alpha5.PortWithNodeFilters{
//				{
//					Port:        fmt.Sprintf("%d:80", clusterPort),
//					NodeFilters: []string{"loadbalancer"},
//				},
//			},
//			Servers: 1,
//		},
//	)
//)
