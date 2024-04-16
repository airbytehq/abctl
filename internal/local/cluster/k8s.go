package cluster

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sigs.k8s.io/kind/pkg/cluster"
	"time"
)

type Provider string

const (
	DockerDesktop Provider = "docker-desktop"
	Kind          Provider = "kind"
)

const (
	k8sVersion = "v1.29.1"
	kubeconfig = "abctl.kubeconfig"
)

type K8s interface {
	Create(name string) error
	Delete(name string) error
	Exists(name string) bool
}

func New(provider Provider) (K8s, error) {
	switch provider {
	case Kind:
		kubeconfigDir, err := createAbctlDirectory()
		if err != nil {
			return nil, fmt.Errorf("unable to create abctl directory: %w", err)
		}
		return &KindK8s{
			p:          cluster.NewProvider(),
			kubeconfig: filepath.Join(kubeconfigDir, kubeconfig),
		}, nil
	}

	return nil, errors.New("unknown provider")
}

type KindK8s struct {
	p          *cluster.Provider
	kubeconfig string
}

func (k *KindK8s) Create(name string) error {
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

func (k *KindK8s) Delete(name string) error {
	if err := k.p.Delete(name, k.kubeconfig); err != nil {
		return fmt.Errorf("unable to delete kind cluster: %w", err)
	}

	return nil
}

func (k *KindK8s) Exists(name string) bool {
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

// createAbctlDirectory creates the ~/.airbyte/abctl directory
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
