package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// MkDirs creates the directories for this providers kubeconfig.
// The kubeconfigs are always scoped to the user's home directory.
// TODO: rename to something more clear
func (p Provider) MkDirs(userHome string) error {
	const permissions = 0700
	kubeconfig := filepath.Join(userHome, p.Kubeconfig)
	if err := os.MkdirAll(filepath.Dir(kubeconfig), permissions); err != nil {
		return fmt.Errorf("could not create directory %s: %v", kubeconfig, err)
	}

	return nil
}

var (
	// DockerDesktopProvider represents the docker-desktop provider.
	DockerDesktopProvider = Provider{
		Name:        "docker-desktop",
		ClusterName: "docker-desktop",
		Context:     "docker-desktop",
		Kubeconfig:  filepath.Join(".kube", "config"),
		HelmNginx:   []string{},
	}

	// KindProvider represents the kind (https://kind.sigs.k8s.io/) provider.
	KindProvider = Provider{
		Name:        "kind",
		ClusterName: "airbyte-abctl",
		Context:     "kind-airbyte-abctl",
		Kubeconfig:  filepath.Join(".airbyte", "abctl", "abctl.kubeconfig"),
		HelmNginx: []string{
			"controller.hostPort.enabled=true",
			"controller.service.type=NodePort",
		},
	}

	// TestProvider represents a test provider, for testing purposes
	TestProvider = Provider{
		Name:        "test",
		ClusterName: "test",
		Context:     "test-abctl",
		Kubeconfig:  filepath.Join(os.TempDir(), "abctl.kubeconfig"),
		HelmNginx:   []string{},
	}
)

// ProviderFromString returns a provider from the given string s.
// If no provider is found, an error is returned.
func ProviderFromString(s string) (Provider, error) {
	switch strings.ToLower(s) {
	case "docker-desktop":
		return DockerDesktopProvider, nil
	case "kind":
		return KindProvider, nil
	case "test":
		return TestProvider, nil
	}

	return Provider{}, fmt.Errorf("unknown provider: %s", s)
}