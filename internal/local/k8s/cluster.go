package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"sigs.k8s.io/kind/pkg/cluster"
	"time"
)

const (
	ClusterName = "airbyte-abctl"
	k8sVersion  = "v1.29.1"
)

// Cluster is an interface representing all the actions taken at the cluster level.
type Cluster interface {
	// Create a cluster with the provided name.
	Create(name string) error
	// Delete a cluster with the provided name.
	Delete(name string) error
	// Exists returns true if the cluster exists, false otherwise.
	Exists(name string) bool
}

// NewCluster returns a Cluster implementation for the provider.
func NewCluster(provider Provider) (Cluster, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine user home directory: %w", err)
	}

	if err := provider.MkDirs(userHome); err != nil {
		return nil, fmt.Errorf("could not create directory: %w", err)
	}

	switch provider.Name {
	case DockerDesktopProvider.Name:
		return &DockerDesktopCluster{
			kubeconfig: filepath.Join(userHome, provider.Kubeconfig),
		}, nil
	case KindProvider.Name:
		return &KindCluster{
			p:          cluster.NewProvider(),
			kubeconfig: filepath.Join(userHome, provider.Kubeconfig),
		}, nil
	}

	return nil, fmt.Errorf("unsupported provider %s", provider)
}

// interface sanity check
var _ Cluster = (*DockerDesktopCluster)(nil)

// DockerDesktopCluster is a Cluster that represents a docker-desktop cluster
type DockerDesktopCluster struct {
	kubeconfig string
}

func (d DockerDesktopCluster) Create(name string) error {
	return nil
}

func (d DockerDesktopCluster) Delete(name string) error {
	return nil
}

func (d DockerDesktopCluster) Exists(name string) bool {
	return true
}

// interface sanity check
var _ Cluster = (*KindCluster)(nil)

// KindCluster is a Cluster implementation for kind (https://kind.sigs.k8s.io/).
type KindCluster struct {
	// p is the kind provider, not the abctl provider
	p *cluster.Provider
	// kubeconfig is the full path to the kubeconfig file kind is using
	kubeconfig string
}

func (k *KindCluster) Create(name string) error {
	// see https://kind.sigs.k8s.io/docs/user/ingress/#create-cluster
	rawCfg := `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
    - |
      kind: InitConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          node-labels: "ingress-ready=true"
    extraPortMappings:
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      - containerPort: 443
        hostPort: 443
        protocol: TCP`

	opts := []cluster.CreateOption{
		cluster.CreateWithWaitForReady(120 * time.Second),
		cluster.CreateWithKubeconfigPath(k.kubeconfig),
		cluster.CreateWithNodeImage("kindest/node:" + k8sVersion),
		cluster.CreateWithRawConfig([]byte(rawCfg)),
	}

	if err := k.p.Create(name, opts...); err != nil {
		return fmt.Errorf("unable to create kind cluster: %w", err)
	}

	return nil
}

func (k *KindCluster) Delete(name string) error {
	if err := k.p.Delete(name, k.kubeconfig); err != nil {
		return fmt.Errorf("unable to delete kind cluster: %w", err)
	}

	return nil
}

func (k *KindCluster) Exists(name string) bool {
	clusters, _ := k.p.List()
	for _, c := range clusters {
		if c == name {
			return true
		}
	}

	return false
}

// permissions sets the file and directory permission level for the kind kube config file and directory.
const permissions = 0700

// createAbctlDirectory creates the ~/.airbyte/abctl directory, if it doesn't already exist.
// If successful returns the full path to the ~/.airbyte/abctl directory
func createAbctlDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	path := filepath.Join(home, ".airbyte", "abctl")
	if err := os.MkdirAll(path, permissions); err != nil {
		return "", fmt.Errorf("could not create abctl directory: %w", err)
	}

	return path, nil
}
