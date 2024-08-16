package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// DefaultPersistentVolumeSize is the size of the disks created by the persistent-volumes and requested by
// the persistent-volume-claims.
var DefaultPersistentVolumeSize = resource.MustParse("500Mi")

// Client primarily for testing purposes
type Client interface {
	// DeploymentRestart will force a restart of the deployment name in the provided namespace.
	// This is a blocking call, it should only return once the deployment has completed.
	DeploymentRestart(ctx context.Context, namespace, name string) error
	// IngressCreate creates an ingress in the given namespace
	IngressCreate(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error
	// IngressExists returns true if the ingress exists in the namespace, false otherwise.
	IngressExists(ctx context.Context, namespace string, ingress string) bool
	// IngressUpdate updates an existing ingress in the given namespace
	IngressUpdate(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error

	// NamespaceCreate creates a namespace
	NamespaceCreate(ctx context.Context, namespace string) error
	// NamespaceExists returns true if the namespace exists, false otherwise
	NamespaceExists(ctx context.Context, namespace string) bool
	// NamespaceDelete deletes the existing namespace
	NamespaceDelete(ctx context.Context, namespace string) error

	// PersistentVolumeCreate creates a persistent volume
	PersistentVolumeCreate(ctx context.Context, namespace, name string) error
	// PersistentVolumeExists returns true if the persistent volume exists, false otherwise
	PersistentVolumeExists(ctx context.Context, namespace, name string) bool
	// PersistentVolumeDelete deletes the existing persistent volume
	PersistentVolumeDelete(ctx context.Context, namespace, name string) error

	// PersistentVolumeClaimCreate creates a persistent volume claim
	PersistentVolumeClaimCreate(ctx context.Context, namespace, name, volumeName string) error
	// PersistentVolumeClaimExists returns true if the persistent volume claim exists, false otherwise
	PersistentVolumeClaimExists(ctx context.Context, namespace, name, volumeName string) bool
	// PersistentVolumeClaimDelete deletes the existing persistent volume claim
	PersistentVolumeClaimDelete(ctx context.Context, namespace, name, volumeName string) error

	// SecretCreateOrUpdate will update or create the secret name with the payload of data in the specified namespace
	SecretCreateOrUpdate(ctx context.Context, secret corev1.Secret) error
	SecretGet(ctx context.Context, namespace, name string) (*corev1.Secret, error)

	// ServiceGet returns the service for the given namespace and name
	ServiceGet(ctx context.Context, namespace, name string) (*corev1.Service, error)

	// ServerVersionGet returns the kubernetes version.
	ServerVersionGet() (string, error)

	EventsWatch(ctx context.Context, namespace string) (watch.Interface, error)

	LogsGet(ctx context.Context, namespace string, name string) (string, error)
}

var _ Client = (*DefaultK8sClient)(nil)

// DefaultK8sClient converts the official kubernetes client to our more manageable (and testable) interface
type DefaultK8sClient struct {
	ClientSet *kubernetes.Clientset
}

func (d *DefaultK8sClient) DeploymentRestart(ctx context.Context, namespace, name string) error {
	restartedAtName := "kubectl.kubernetes.io/restartedAt"
	restartedAtValue := time.Now().Format(time.RFC3339)

	// similar to how kubectl rollout restart works, patch in a restartedAt annotation.
	rawPatch := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{
						restartedAtName: restartedAtValue,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(rawPatch)
	if err != nil {
		return fmt.Errorf("unable to marshal raw patch: %w", err)
	}

	deployment, err := d.ClientSet.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, jsonData, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to fetch deployment %s: %w", name, err)
	}

	label := metav1.FormatLabelSelector(deployment.Spec.Selector)

	deploymentPods := func(ctx context.Context) (bool, error) {
		pods, err := d.ClientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: label,
		})
		if err != nil {
			return false, fmt.Errorf("unable to list pods for deployment %s: %w", name, err)
		}

		for _, pod := range pods.Items {
			// if any pods are not running or are missing the restartedAt annotation
			// then the restart isn't complete
			if pod.Status.Phase != corev1.PodRunning || pod.ObjectMeta.Annotations[restartedAtName] != restartedAtValue {
				return false, nil
			}

			// even though a pod is running, doesn't mean it is ready
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
					return false, nil
				}
			}
		}

		// if we're here, then all the pods are running with the correct restartedAt annotation
		// and they're in a ready state
		return true, nil
	}

	// check every 10 seconds for up to 5 minutes to see if the pods have been restarted successfully
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, deploymentPods)
	if err != nil {
		return fmt.Errorf("unable to restart deployment %s: %w", name, err)
	}

	return nil
}

func (d *DefaultK8sClient) IngressCreate(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	_, err := d.ClientSet.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
	return err
}

func (d *DefaultK8sClient) IngressExists(ctx context.Context, namespace string, ingress string) bool {
	_, err := d.ClientSet.NetworkingV1().Ingresses(namespace).Get(ctx, ingress, metav1.GetOptions{})
	if err == nil {
		return true
	}

	return !k8serrors.IsNotFound(err)
}

func (d *DefaultK8sClient) IngressUpdate(ctx context.Context, namespace string, ingress *networkingv1.Ingress) error {
	_, err := d.ClientSet.NetworkingV1().Ingresses(namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	return err
}

func (d *DefaultK8sClient) NamespaceCreate(ctx context.Context, namespace string) error {
	_, err := d.ClientSet.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{})
	return err
}

func (d *DefaultK8sClient) NamespaceExists(ctx context.Context, namespace string) bool {
	_, err := d.ClientSet.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return true
	}

	return !k8serrors.IsNotFound(err)
}

func (d *DefaultK8sClient) NamespaceDelete(ctx context.Context, namespace string) error {
	return d.ClientSet.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
}

func (d *DefaultK8sClient) PersistentVolumeCreate(ctx context.Context, namespace, name string) error {
	hostPathType := corev1.HostPathDirectoryOrCreate

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{corev1.ResourceStorage: DefaultPersistentVolumeSize},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: path.Join("/var/local-path-provisioner", name),
					Type: &hostPathType,
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			PersistentVolumeReclaimPolicy: "Retain",
			StorageClassName:              "standard",
		},
	}

	_, err := d.ClientSet.CoreV1().PersistentVolumes().Create(ctx, pv, metav1.CreateOptions{})
	return err
}

func (d *DefaultK8sClient) PersistentVolumeExists(ctx context.Context, _, name string) bool {
	_, err := d.ClientSet.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return true
	}
	return !k8serrors.IsNotFound(err)
}

func (d *DefaultK8sClient) PersistentVolumeDelete(ctx context.Context, _, name string) error {
	return d.ClientSet.CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{})
}

func (d *DefaultK8sClient) PersistentVolumeClaimCreate(ctx context.Context, namespace, name, volumeName string) error {
	storageClass := "standard"

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: DefaultPersistentVolumeSize}},
			VolumeName:       volumeName,
			StorageClassName: &storageClass,
		},
		Status: corev1.PersistentVolumeClaimStatus{},
	}

	_, err := d.ClientSet.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
	return err
}

func (d *DefaultK8sClient) PersistentVolumeClaimExists(ctx context.Context, namespace, name, _ string) bool {
	_, err := d.ClientSet.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return true
	}

	return !k8serrors.IsNotFound(err)
}

func (d *DefaultK8sClient) PersistentVolumeClaimDelete(ctx context.Context, namespace, name, _ string) error {
	return d.ClientSet.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (d *DefaultK8sClient) SecretCreateOrUpdate(ctx context.Context, secret corev1.Secret) error {
	namespace := secret.ObjectMeta.Namespace
	name := secret.ObjectMeta.Name
	_, err := d.ClientSet.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		// update
		if _, err := d.ClientSet.CoreV1().Secrets(namespace).Update(ctx, &secret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("unable to update the secret %s: %w", name, err)
		}

		return nil
	}

	if k8serrors.IsNotFound(err) {
		if _, err := d.ClientSet.CoreV1().Secrets(namespace).Create(ctx, &secret, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("unable to create the secret %s: %w", name, err)
		}

		return nil
	}

	return fmt.Errorf("unexpected error while handling the secret %s: %w", name, err)
}

func (d *DefaultK8sClient) SecretGet(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	secret, err := d.ClientSet.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get the secret %s: %w", name, err)
	}
	return secret, nil
}

func (d *DefaultK8sClient) ServerVersionGet() (string, error) {
	ver, err := d.ClientSet.DiscoveryClient.ServerVersion()
	if err != nil {
		return "", err
	}

	return ver.String(), nil
}

func (d *DefaultK8sClient) ServiceGet(ctx context.Context, namespace string, name string) (*corev1.Service, error) {
	return d.ClientSet.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (d *DefaultK8sClient) EventsWatch(ctx context.Context, namespace string) (watch.Interface, error) {
	return d.ClientSet.EventsV1().Events(namespace).Watch(ctx, metav1.ListOptions{})
}

func (d *DefaultK8sClient) LogsGet(ctx context.Context, namespace string, name string) (string, error) {
	req := d.ClientSet.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{})
	reader, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to get logs for pod %s: %w", name, err)
	}
	defer reader.Close()
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, reader); err != nil {
		return "", fmt.Errorf("unable to copy logs from pod %s: %w", name, err)
	}
	return buf.String(), nil
}
