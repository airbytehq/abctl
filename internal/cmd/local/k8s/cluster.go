package k8s

import (
	"fmt"
	"os"
	"time"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s/kind"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/cluster"
)

// ExtraVolumeMount defines a host volume mount for the Kind cluster
type ExtraVolumeMount struct {
	HostPath      string
	ContainerPath string
}

// Cluster is an interface representing all the actions taken at the cluster level.
type Cluster interface {
	// Create a cluster with the provided name.
	Create(portHTTP int, extraMounts []ExtraVolumeMount) error
	// Delete a cluster with the provided name.
	Delete() error
	// Exists returns true if the cluster exists, false otherwise.
	Exists() bool
}

// interface sanity check
var _ Cluster = (*kindCluster)(nil)

// kindCluster is a Cluster implementation for kind (https://kind.sigs.k8s.io/).
type kindCluster struct {
	// p is the kind provider, not the abctl provider
	p *cluster.Provider
	// kubeconfig is the full path to the kubeconfig file kind is using
	kubeconfig  string
	clusterName string
}

// k8sVersion is the kind node version being used.
// Note that the sha256 must match the version listed on the release for the specific version of kind
// that we're currently using (e.g. https://github.com/kubernetes-sigs/kind/releases/tag/v0.23.0)
const k8sVersion = "v1.29.4@sha256:3abb816a5b1061fb15c6e9e60856ec40d56b7b52bcea5f5f1350bc6e2320b6f8"

func (k *kindCluster) Create(port int, extraMounts []ExtraVolumeMount) error {
	// Create the data directory before the cluster does to ensure that it's owned by the correct user.
	// If the cluster creates it and docker is running as root, it's possible that root will own this directory
	// which will cause minio and postgres to break.
	pterm.Debug.Println(fmt.Sprintf("Creating data directory '%s'", paths.Data))
	if err := os.MkdirAll(paths.Data, 0766); err != nil {
		pterm.Error.Println(fmt.Sprintf("Error creating data directory '%s'", paths.Data))
		return fmt.Errorf("unable to create directory '%s': %w", paths.Data, err)
	}

	// see https://kind.sigs.k8s.io/docs/user/ingress/#create-cluster
	config := kind.DefaultConfig().WithHostPort(port)
	for _, mount := range extraMounts {
		config = config.WithVolumeMount(mount.HostPath, mount.ContainerPath)
	}

	rawCfg, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("unable to marshal Kind cluster config: %w", err)
	}

	opts := []cluster.CreateOption{
		cluster.CreateWithWaitForReady(5 * time.Minute),
		cluster.CreateWithKubeconfigPath(k.kubeconfig),
		cluster.CreateWithNodeImage("kindest/node:" + k8sVersion),
		cluster.CreateWithRawConfig(rawCfg),
	}

	if err := k.p.Create(k.clusterName, opts...); err != nil {
		return fmt.Errorf("unable to create kind cluster: %w", err)
	}

	return nil
}

func (k *kindCluster) Delete() error {
	if err := k.p.Delete(k.clusterName, k.kubeconfig); err != nil {
		return fmt.Errorf("unable to delete kind cluster: %w", err)
	}

	return nil
}

func (k *kindCluster) Exists() bool {
	clusters, _ := k.p.List()
	for _, c := range clusters {
		if c == k.clusterName {
			return true
		}
	}

	return false
}
