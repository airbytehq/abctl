package k8s

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sigs.k8s.io/kind/pkg/cluster"
	"time"
)

const (
	k8sVersion = "v1.29.1"
	kubeconfig = "abctl.kubeconfig"
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

// New returns a Cluster implementation for the provider.
func New(provider Provider) (Cluster, error) {
	switch provider {
	case Kind:
		kubeconfigDir, err := createAbctlDirectory()
		if err != nil {
			return nil, fmt.Errorf("unable to create abctl directory: %w", err)
		}
		return &KindCluster{
			p:          cluster.NewProvider(),
			kubeconfig: filepath.Join(kubeconfigDir, kubeconfig),
		}, nil
	}

	return nil, errors.New("unknown provider")
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
	opts := []cluster.CreateOption{
		cluster.CreateWithWaitForReady(120 * time.Second),
		cluster.CreateWithKubeconfigPath(k.kubeconfig),
		cluster.CreateWithNodeImage("kindest/node:" + k8sVersion),
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
