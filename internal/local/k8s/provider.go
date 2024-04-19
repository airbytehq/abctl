package k8s

import (
	"fmt"
	"os"
	"path/filepath"
)

type Provider struct {
	Name       string
	kubeconfig string
}

// Kubeconfig returns the kubeconfig for this provider.
// The kubeconfigs are always scoped to the user's home directory.
func (p Provider) Kubeconfig(userHome string) string {
	return filepath.Join(userHome, p.kubeconfig)
}

// MkDirs creates the directories for this providers kubeconfig.
// The kubeconfigs are always scoped to the user's home directory.
// TODO: rename to something more clear
func (p Provider) MkDirs(userHome string) error {
	const permissions = 0700
	if err := os.MkdirAll(p.Kubeconfig(userHome), permissions); err != nil {
		return fmt.Errorf("could not create directory %s: %v", p.Kubeconfig(userHome), err)
	}

	return nil
}

// String returns a human-readable name of this provider.
func (p Provider) String() string {
	return p.Name
}

var (
	// DockerDesktop represents the docker-desktop provider.
	DockerDesktop = Provider{
		Name:       "docker-desktop",
		kubeconfig: filepath.Join(".kube", "config"),
	}
	// Kind represents the kind (https://kind.sigs.k8s.io/) provider.
	Kind = Provider{
		Name:       "kind",
		kubeconfig: filepath.Join(".airbyte", "abctl", "abctl.kubeconfig"),
	}
)

func ProviderFromString(s string) (Provider, error) {
	switch s {
	case "docker-desktop":
		return DockerDesktop, nil
	case "kind":
		return Kind, nil
	}

	return Provider{}, fmt.Errorf("unknown provider: %s", s)
}
