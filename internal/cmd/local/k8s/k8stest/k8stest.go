package k8stest

import (
	"context"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var _ k8s.Client = (*MockClient)(nil)

type MockClient struct {
	FnDeploymentList              func(ctx context.Context, namespace string) (*v1.DeploymentList, error)
	FnDeploymentRestart           func(ctx context.Context, namespace, name string) error
	FnIngressCreate               func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error
	FnIngressExists               func(ctx context.Context, namespace string, ingress string) bool
	FnIngressUpdate               func(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error
	FnNamespaceCreate             func(ctx context.Context, namespace string) error
	FnNamespaceExists             func(ctx context.Context, namespace string) bool
	FnNamespaceDelete             func(ctx context.Context, namespace string) error
	FnPersistentVolumeCreate      func(ctx context.Context, namespace, name string) error
	FnPersistentVolumeExists      func(ctx context.Context, namespace, name string) bool
	FnPersistentVolumeDelete      func(ctx context.Context, namespace, name string) error
	FnPersistentVolumeClaimCreate func(ctx context.Context, namespace, name, volumeName string) error
	FnPersistentVolumeClaimExists func(ctx context.Context, namespace, name, volumeName string) bool
	FnPersistentVolumeClaimDelete func(ctx context.Context, namespace, name, volumeName string) error
	FnSecretCreateOrUpdate        func(ctx context.Context, secret corev1.Secret) error
	FnSecretGet                   func(ctx context.Context, namespace, name string) (*corev1.Secret, error)
	FnServerVersionGet            func() (string, error)
	FnServiceGet                  func(ctx context.Context, namespace, name string) (*corev1.Service, error)
	FnEventsWatch                 func(ctx context.Context, namespace string) (watch.Interface, error)
	FnLogsGet                     func(ctx context.Context, namespace string, name string) (string, error)
}

func (m *MockClient) DeploymentList(ctx context.Context, namespace string) (*v1.DeploymentList, error) {
	return m.FnDeploymentList(ctx, namespace)
}

func (m *MockClient) DeploymentRestart(ctx context.Context, namespace, name string) error {
	if m.FnDeploymentRestart == nil {
		return m.FnDeploymentRestart(ctx, namespace, name)
	}
	return nil
}

func (m *MockClient) IngressCreate(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	if m.FnIngressCreate != nil {
		return m.FnIngressCreate(ctx, namespace, ingress)
	}
	return nil
}

func (m *MockClient) IngressExists(ctx context.Context, namespace string, ingress string) bool {
	if m.FnIngressExists != nil {
		return m.FnIngressExists(ctx, namespace, ingress)
	}
	return true
}

func (m *MockClient) IngressUpdate(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	if m.FnIngressUpdate != nil {
		return m.FnIngressUpdate(ctx, namespace, ingress)
	}
	return nil
}

func (m *MockClient) NamespaceCreate(ctx context.Context, namespace string) error {
	if m.FnNamespaceCreate != nil {
		return m.FnNamespaceCreate(ctx, namespace)
	}
	return nil
}

func (m *MockClient) NamespaceExists(ctx context.Context, namespace string) bool {
	if m.FnNamespaceExists != nil {
		return m.FnNamespaceExists(ctx, namespace)
	}
	return true
}

func (m *MockClient) NamespaceDelete(ctx context.Context, namespace string) error {
	if m.FnNamespaceDelete != nil {
		return m.FnNamespaceDelete(ctx, namespace)
	}
	return nil
}

func (m *MockClient) PersistentVolumeCreate(ctx context.Context, namespace, name string) error {
	if m.FnPersistentVolumeCreate != nil {
		return m.FnPersistentVolumeCreate(ctx, namespace, name)
	}
	return nil
}
func (m *MockClient) PersistentVolumeExists(ctx context.Context, namespace, name string) bool {
	if m.FnPersistentVolumeExists != nil {
		return m.FnPersistentVolumeExists(ctx, namespace, name)
	}
	return true
}
func (m *MockClient) PersistentVolumeDelete(ctx context.Context, namespace, name string) error {
	if m.FnPersistentVolumeDelete != nil {
		return m.FnPersistentVolumeDelete(ctx, namespace, name)
	}
	return nil
}

func (m *MockClient) PersistentVolumeClaimCreate(ctx context.Context, namespace, name, volumeName string) error {
	if m.FnPersistentVolumeClaimCreate != nil {
		return m.FnPersistentVolumeClaimCreate(ctx, namespace, name, volumeName)
	}
	return nil
}
func (m *MockClient) PersistentVolumeClaimExists(ctx context.Context, namespace, name, volumeName string) bool {
	if m.FnPersistentVolumeClaimExists != nil {
		return m.FnPersistentVolumeClaimExists(ctx, namespace, name, volumeName)
	}
	return true
}
func (m *MockClient) PersistentVolumeClaimDelete(ctx context.Context, namespace, name, volumeName string) error {
	if m.FnPersistentVolumeClaimDelete != nil {
		return m.FnPersistentVolumeClaimDelete(ctx, namespace, name, volumeName)
	}
	return nil
}

func (m *MockClient) SecretCreateOrUpdate(ctx context.Context, secret corev1.Secret) error {
	if m.FnSecretCreateOrUpdate != nil {
		return m.FnSecretCreateOrUpdate(ctx, secret)
	}

	return nil
}

func (m *MockClient) SecretGet(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	if m.FnSecretGet != nil {
		return m.FnSecretGet(ctx, namespace, name)
	}

	return nil, nil
}

func (m *MockClient) ServiceGet(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	return m.FnServiceGet(ctx, namespace, name)
}

func (m *MockClient) ServerVersionGet() (string, error) {
	if m.FnServerVersionGet != nil {
		return m.FnServerVersionGet()
	}
	return "test", nil
}

func (m *MockClient) EventsWatch(ctx context.Context, namespace string) (watch.Interface, error) {
	if m.FnEventsWatch == nil {
		return watch.NewFake(), nil
	}
	return m.FnEventsWatch(ctx, namespace)
}

func (m *MockClient) LogsGet(ctx context.Context, namespace string, name string) (string, error) {
	if m.FnLogsGet == nil {
		return "LogsGet called", nil
	}
	return m.FnLogsGet(ctx, namespace, name)
}
