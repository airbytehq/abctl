package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"sigs.k8s.io/kind/pkg/cluster"
)

const (
	Kind = "kind"
	Test = "test"
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

// mkDirs creates the directories for this providers kubeconfig.
// The kubeconfigs are always scoped to the user's home directory.
func (p Provider) mkDirs(userHome string) error {
	const permissions = 0700
	kubeconfig := filepath.Join(userHome, p.Kubeconfig)
	if err := os.MkdirAll(filepath.Dir(kubeconfig), permissions); err != nil {
		return fmt.Errorf("could not create directory %s: %v", kubeconfig, err)
	}

	return nil
}

// Cluster returns a kubernetes cluster for this provider.
func (p Provider) Cluster() (Cluster, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine user home directory: %w", err)
	}

	if err := p.mkDirs(home); err != nil {
		return nil, fmt.Errorf("could not create directory %s: %w", home, err)
	}

	return &kindCluster{
		p:           cluster.NewProvider(),
		kubeconfig:  filepath.Join(home, p.Kubeconfig),
		clusterName: p.ClusterName,
	}, nil
}

var (
	// DefaultProvider represents the kind (https://kind.sigs.k8s.io/) provider.
	DefaultProvider = Provider{
		Name:        Kind,
		ClusterName: "airbyte-abctl",
		Context:     "kind-airbyte-abctl",
		Kubeconfig:  filepath.Join(".airbyte", "abctl", "abctl.kubeconfig"),
		HelmNginx: []string{
			"controller.hostPort.enabled=true",
			"controller.service.httpsPort.enable=false",
			"controller.service.type=NodePort",
		},
	}

	// TestProvider represents a test provider, for testing purposes
	TestProvider = Provider{
		Name:        Test,
		ClusterName: "test",
		Context:     "test-abctl",
		Kubeconfig:  filepath.Join(os.TempDir(), "abctl.kubeconfig"),
		HelmNginx:   []string{},
	}
)
