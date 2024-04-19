package k8s

import (
	"errors"
	"path/filepath"
)

// Provider is a representation of which k8s providers are currently supported.
type Provider string

func (p Provider) Kubeconfig(userHome string) (string, error) {
	switch p {
	case DockerDesktop:
		return filepath.Join(userHome, ".kube", "config"), nil
	case Kind:
		return filepath.Join(userHome, ".airbyte", "abctl", "abctl.kubeconfig"), nil
	}

	return "", errors.New("unknown provider " + p.String())
}

func (p Provider) String() string {
	return string(p)
}

const (
	DockerDesktop Provider = "docker-desktop"
	Kind          Provider = "kind"
)
