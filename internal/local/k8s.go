package local

import (
	"context"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sClient primarily for testing purposes
type K8sClient interface {
	// CreateIngress creates an ingress in the given namespace
	CreateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error
	// ExistsIngress returns true if the ingress exists in the namespace, false otherwise.
	ExistsIngress(ctx context.Context, namespace string, ingress string) bool
	// UpdateIngress updates an existing ingress in the given namespace
	UpdateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error

	// ExistsNamespace returns true if the namespace exists, false otherwise
	ExistsNamespace(ctx context.Context, namespace string) bool
	// DeleteNamespace deletes the existing namespace
	DeleteNamespace(ctx context.Context, namespace string) error

	// GetServerVersion returns the k8s version.
	GetServerVersion() (string, error)
}

// defaultK8sClient converts the official kubernetes client to our more manageable interface
type defaultK8sClient struct {
	k8s *kubernetes.Clientset
}

func (d *defaultK8sClient) CreateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	_, err := d.k8s.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, v1.CreateOptions{})
	return err
}

func (d *defaultK8sClient) ExistsIngress(ctx context.Context, namespace string, ingress string) bool {
	_, err := d.k8s.NetworkingV1().Ingresses(namespace).Get(ctx, ingress, v1.GetOptions{})
	if err == nil {
		return true
	}

	return !k8serrors.IsNotFound(err)
}

func (d *defaultK8sClient) UpdateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	_, err := d.k8s.NetworkingV1().Ingresses(namespace).Update(ctx, ingress, v1.UpdateOptions{})
	return err
}

func (d *defaultK8sClient) ExistsNamespace(ctx context.Context, namespace string) bool {
	_, err := d.k8s.CoreV1().Namespaces().Get(ctx, namespace, v1.GetOptions{})
	if err == nil {
		return true
	}

	return !k8serrors.IsNotFound(err)
}

func (d *defaultK8sClient) DeleteNamespace(ctx context.Context, namespace string) error {
	return d.k8s.CoreV1().Namespaces().Delete(ctx, namespace, v1.DeleteOptions{})
}

func (d *defaultK8sClient) GetServerVersion() (string, error) {
	ver, err := d.k8s.DiscoveryClient.ServerVersion()
	if err != nil {
		return "", err
	}

	return ver.String(), nil
}
