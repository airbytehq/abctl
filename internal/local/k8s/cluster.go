package k8s

import (
	"fmt"
	"github.com/airbytehq/abctl/internal/local/localerr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"sigs.k8s.io/kind/pkg/cluster"
	"time"
)

const (
	k8sVersion = "v1.29.1"
)

// Cluster is an interface representing all the actions taken at the cluster level.
type Cluster interface {
	// Create a cluster with the provided name.
	Create(portHTTP int) error
	// Delete a cluster with the provided name.
	Delete() error
	// Exists returns true if the cluster exists, false otherwise.
	Exists() bool
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
		kubeCfg := filepath.Join(userHome, provider.Kubeconfig)
		return &DockerDesktopCluster{
			clientCfg: clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeCfg},
				&clientcmd.ConfigOverrides{CurrentContext: provider.Context},
			),
			kubeCfg:     filepath.Join(userHome, provider.Kubeconfig),
			clusterName: provider.ClusterName,
		}, nil
	case KindProvider.Name:
		return &KindCluster{
			p:           cluster.NewProvider(),
			kubeconfig:  filepath.Join(userHome, provider.Kubeconfig),
			clusterName: provider.ClusterName,
		}, nil
	}

	return nil, fmt.Errorf("unsupported provider %s", provider)
}

// interface sanity check
var _ Cluster = (*DockerDesktopCluster)(nil)

// DockerDesktopCluster is a Cluster that represents a docker-desktop cluster
type DockerDesktopCluster struct {
	clientCfg   clientcmd.ClientConfig
	kubeCfg     string
	clusterName string
}

func (d DockerDesktopCluster) Create(_ int) error {
	return fmt.Errorf("%w: docker-desktop cluster should already exist", localerr.ErrKubernetes)
}

func (d DockerDesktopCluster) Delete() error {
	return nil
}

func (d DockerDesktopCluster) Exists() bool {
	restCfg, err := d.clientCfg.ClientConfig()
	if err != nil {
		return false
	}
	cli, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return false
	}
	v, err := cli.ServerVersion()
	if err != nil || v.Platform == "" {
		return false
	}

	return true
}

// interface sanity check
var _ Cluster = (*KindCluster)(nil)

// KindCluster is a Cluster implementation for kind (https://kind.sigs.k8s.io/).
type KindCluster struct {
	// p is the kind provider, not the abctl provider
	p *cluster.Provider
	// kubeconfig is the full path to the kubeconfig file kind is using
	kubeconfig  string
	clusterName string
}

func (k *KindCluster) Create(portHTTP int) error {
	// see https://kind.sigs.k8s.io/docs/user/ingress/#create-cluster
	rawCfg := fmt.Sprintf(`kind: Cluster
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
        hostPort: %d
        protocol: TCP`,
		portHTTP)
	//- containerPort: 443
	//  hostPort: 443
	//  protocol: TCP`

	opts := []cluster.CreateOption{
		cluster.CreateWithWaitForReady(120 * time.Second),
		cluster.CreateWithKubeconfigPath(k.kubeconfig),
		cluster.CreateWithNodeImage("kindest/node:" + k8sVersion),
		cluster.CreateWithRawConfig([]byte(rawCfg)),
	}

	if err := k.p.Create(k.clusterName, opts...); err != nil {
		return fmt.Errorf("unable to create kind cluster: %w", err)
	}

	return nil
}

func (k *KindCluster) Delete() error {
	if err := k.p.Delete(k.clusterName, k.kubeconfig); err != nil {
		return fmt.Errorf("unable to delete kind cluster: %w", err)
	}

	return nil
}

func (k *KindCluster) Exists() bool {
	clusters, _ := k.p.List()
	for _, c := range clusters {
		if c == k.clusterName {
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
