package k8s

import (
	"context"
	"fmt"
)

// ClusterFactory creates k8s clusters
type ClusterFactory func(ctx context.Context, clusterName string) (Cluster, error)

// DefaultClusterFactory creates a default cluster for the given cluster name
func DefaultClusterFactory(ctx context.Context, clusterName string) (Cluster, error) {
	provider := Provider{
		Name:        Kind,
		ClusterName: clusterName,
		Context:     fmt.Sprintf("kind-%s", clusterName),
		Kubeconfig:  DefaultProvider.Kubeconfig,
	}
	return provider.Cluster(ctx)
}
