package k8s

import (
	"fmt"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"os"
	"path/filepath"
	"sigs.k8s.io/kind/pkg/cluster"
)

// Provider represents a k8s provider.
type Provider struct {
	// Name of this provider
	Name string
	// ClusterName is the name of the cluster this provider will interact with
	ClusterName string
	// Context this provider should use
	Context string
	// Kubeconfig location
	Kubeconfig string
	// HelmNginx additional helm values to pass to the nginx chart
	HelmNginx []string
}

// Cluster returns a kubernetes cluster for this provider.
func (p Provider) Cluster() (Cluster, error) {
	if err := os.MkdirAll(filepath.Dir(p.Kubeconfig), 766); err != nil {
		return nil, fmt.Errorf("unable to create directory %s: %v", p.Kubeconfig, err)
	}

	return &kindCluster{
		p:           cluster.NewProvider(),
		kubeconfig:  p.Kubeconfig,
		clusterName: p.ClusterName,
	}, nil
}

const (
	Kind = "kind"
	Test = "test"
)

var (
	// DefaultProvider represents the kind (https://kind.sigs.k8s.io/) provider.
	DefaultProvider = Provider{
		Name:        Kind,
		ClusterName: "airbyte-abctl",
		Context:     "kind-airbyte-abctl",
		Kubeconfig:  paths.Kubeconfig,
		HelmNginx: []string{
			"controller.hostPort.enabled=true",
			"controller.service.httpsPort.enable=false",
			"controller.service.type=NodePort",
		},
	}

	// TestProvider represents a test provider, for testing purposes
	TestProvider = Provider{
		Name:        Test,
		ClusterName: "test-airbyte-abctl",
		Context:     "test-airbyte-abctl",
		Kubeconfig:  filepath.Join(os.TempDir(), "abctl", paths.FileKubeconfig),
		HelmNginx:   []string{},
	}
)
