package helm

import (
	goHelm "github.com/mittwald/go-helm-client"
)

// Factory creates helm clients
type Factory func(kubeConfig, kubeContext, namespace string) (goHelm.Client, error)

// DefaultFactory creates a helm client
func DefaultFactory(kubeConfig, kubeContext, namespace string) (goHelm.Client, error) {
	return New(kubeConfig, kubeContext, namespace)
}
