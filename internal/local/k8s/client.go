package k8s

import (
	"context"
	"fmt"
	coreV1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// CreateOrUpdateSecret will update or create the secret name with the payload of data in the specified namespace
	CreateOrUpdateSecret(ctx context.Context, namespace, name string, data map[string][]byte) error

	// GetServerVersion returns the k8s version.
	GetServerVersion() (string, error)
}

// DefaultK8sClient converts the official kubernetes client to our more manageable (and testable) interface
type DefaultK8sClient struct {
	ClientSet *kubernetes.Clientset
}

func (d *DefaultK8sClient) CreateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	_, err := d.ClientSet.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
	return err
}

func (d *DefaultK8sClient) ExistsIngress(ctx context.Context, namespace string, ingress string) bool {
	_, err := d.ClientSet.NetworkingV1().Ingresses(namespace).Get(ctx, ingress, metav1.GetOptions{})
	if err == nil {
		return true
	}

	return !k8serrors.IsNotFound(err)
}

func (d *DefaultK8sClient) UpdateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	_, err := d.ClientSet.NetworkingV1().Ingresses(namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	return err
}

func (d *DefaultK8sClient) ExistsNamespace(ctx context.Context, namespace string) bool {
	_, err := d.ClientSet.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return true
	}

	return !k8serrors.IsNotFound(err)
}

func (d *DefaultK8sClient) DeleteNamespace(ctx context.Context, namespace string) error {
	return d.ClientSet.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
}

func (d *DefaultK8sClient) CreateOrUpdateSecret(ctx context.Context, namespace, name string, data map[string][]byte) error {
	secret := &coreV1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: data,
	}
	_, err := d.ClientSet.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		// update
		if _, err := d.ClientSet.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("could not update the secret %s: %w", name, err)
		}

		return nil
	}

	if k8serrors.IsNotFound(err) {
		if _, err := d.ClientSet.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("could not create the secret %s: %w", name, err)
		}

		return nil
	}

	return fmt.Errorf("unexpected error while handling the secret %s: %w", name, err)
}

func (d *DefaultK8sClient) GetServerVersion() (string, error) {
	ver, err := d.ClientSet.DiscoveryClient.ServerVersion()
	if err != nil {
		return "", err
	}

	return ver.String(), nil
}
