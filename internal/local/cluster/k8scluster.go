package cluster

import (
	"fmt"
	"sigs.k8s.io/kind/pkg/cluster"
	"time"
)

type Provider string

const (
	DockerDesktop Provider = "docker-desktop"
	Kind          Provider = "kind"
)

const (
	K8sVersion = "v1.29.1"
)

type K8sCluster interface {
	Create(name string) error
	Delete(name string) error
	Exists(name string) bool
}

func New(provider Provider) K8sCluster {
	switch provider {
	case Kind:
		return &KindProvider{
			p: cluster.NewProvider(),
		}
	default:
		return nil
	}
}

type KindProvider struct {
	p *cluster.Provider
}

func (k *KindProvider) Create(name string) error {
	opts := []cluster.CreateOption{
		cluster.CreateWithWaitForReady(1 * time.Minute),
		cluster.CreateWithNodeImage("kindest/node:" + K8sVersion),
	}

	if err := k.p.Create(name, opts...); err != nil {
		return fmt.Errorf("unable to create kind cluster: %w", err)
	}

	return nil
}

func (k *KindProvider) Delete(name string) error {
	if err := k.p.Delete(name, ""); err != nil {
		return fmt.Errorf("unable to delete kind cluster: %w", err)
	}

	return nil
}

func (k *KindProvider) Exists(name string) bool {
	clusters, _ := k.p.List()
	for _, c := range clusters {
		if c == name {
			return true
		}
	}

	return false
}
