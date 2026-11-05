package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

type ComponentClient interface {
	ConfigureCloudMCP(ctx context.Context, namespace, release, clientID, clientSecret string) error
}

func NewComponentClient(kubeconfig, kubeContext string) (ComponentClient, error) {
	loading := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loading.ExplicitPath = kubeconfig
	}
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loading, &clientcmd.ConfigOverrides{CurrentContext: kubeContext})
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("component Kubernetes context is unavailable: %w", err)
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("component Kubernetes client creation failed: %w", err)
	}
	return &DefaultK8sClient{ClientSet: client}, nil
}

func (d *DefaultK8sClient) ConfigureCloudMCP(ctx context.Context, namespace, release, clientID, clientSecret string) error {
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("Cloud MCP registration returned incomplete credentials. Retry the installation.")
	}
	secrets := d.ClientSet.CoreV1().Secrets(namespace)
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secret, err := secrets.Get(ctx, "cloud-mcp-oidc", metav1.GetOptions{})
		creating := apierrors.IsNotFound(err)
		if err != nil && !creating {
			return err
		}
		if creating {
			secret = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "cloud-mcp-oidc", Namespace: namespace}, Type: corev1.SecretTypeOpaque}
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data["clientId"] = []byte(clientID)
		secret.Data["clientSecret"] = []byte(clientSecret)
		if creating {
			_, err = secrets.Create(ctx, secret, metav1.CreateOptions{})
		} else {
			_, err = secrets.Update(ctx, secret, metav1.UpdateOptions{})
		}
		return err
	}); err != nil {
		return fmt.Errorf("Cloud MCP credential Secret write failed: %w", err)
	}
	selector := labels.SelectorFromSet(labels.Set{"app": "cloudMcp", "airbyte.io/component-role": "workload", "app.kubernetes.io/instance": release})
	deployments, err := d.ClientSet.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Cloud MCP deployment lookup failed: %w", err)
	}
	var matches []string
	// The component chart labels pod selectors, not Deployment metadata.
	for _, deployment := range deployments.Items {
		if deployment.Spec.Selector != nil && selector.Matches(labels.Set(deployment.Spec.Selector.MatchLabels)) {
			matches = append(matches, deployment.Name)
		}
	}
	if len(matches) != 1 {
		return fmt.Errorf("Cloud MCP requires exactly one deployment in the release; found %d", len(matches))
	}
	patch, err := json.Marshal(map[string]any{"spec": map[string]any{"template": map[string]any{"metadata": map[string]any{"annotations": map[string]string{"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339Nano)}}}}})
	if err != nil {
		return fmt.Errorf("Cloud MCP restart patch cannot be encoded")
	}
	if _, err := d.ClientSet.AppsV1().Deployments(namespace).Patch(ctx, matches[0], types.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("Cloud MCP deployment restart failed: %w", err)
	}
	return nil
}
