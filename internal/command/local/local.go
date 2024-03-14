package local

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	k3d "github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/types"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"
	"os"
)

const (
	chartName   = "airbyte/airbyte"
	clusterName = "airbyte-abctl"
	namespace   = "abctl"
	repoName    = "airbyte"
	repoUrl     = "https://airbytehq.github.io/helm-charts"
)

type Command struct {
	helm helmclient.Client
}

func New() (*Command, error) {
	helm, err := helmclient.New(&helmclient.Options{Namespace: namespace})
	if err != nil {
		return nil, errors.Wrap(err, "could not create helm client")
	}
	return &Command{helm: helm}, nil
}

func (lc *Command) Install() error {
	if err := checkDocker(); err != nil {
		return err
	}

	if err := lc.helm.AddOrUpdateChartRepo(repo.Entry{
		Name: repoName,
		URL:  repoUrl,
	}); err != nil {
		return errors.Wrap(err, "could not add airbyte chart repo")
	}

	fmt.Printf("fetching chart %s...\n", chartName)
	chart, _, err := lc.helm.GetChart(chartName, &action.ChartPathOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not fetch chart %s", chartName))
	}
	fmt.Printf("successfully fetched chart %s\n", chartName)
	fmt.Printf(" version: %s\n", chart.Metadata.Version)
	fmt.Printf(" app-version: %s\n", chart.Metadata.AppVersion)

	fmt.Printf("starting k3d cluster...\n")
	k3dCluster := types.Cluster{
		Name: clusterName,
		KubeAPI: &types.ExposureOpts{
			PortMapping: nat.PortMapping{},
			Host:        types.DefaultAPIHost,
		},
	}

	//x, err := k3d.ClusterGet(context.Background(), runtimes.SelectedRuntime, &k3dCluster)
	if err := k3d.ClusterRun(context.Background(), runtimes.SelectedRuntime, &v1alpha5.ClusterConfig{
		Cluster: k3dCluster,
		ClusterCreateOpts: types.ClusterCreateOpts{
			DisableImageVolume:  false,
			WaitForServer:       false,
			DisableLoadBalancer: false,
			NodeHooks:           nil,
			GlobalLabels:        map[string]string{},
			GlobalEnv:           nil,
			HostAliases:         nil,
		},
	}); err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not create k3d cluster %s", clusterName))
	}
	if err != nil {
		return errors.Wrap(err, "could not communicate with k3d")
	}
	fmt.Println("k3d cluster successfully created")
	//fmt.Println(x)

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
