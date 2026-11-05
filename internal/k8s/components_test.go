package k8s

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func cloudDeployment(name, release string) *appsv1.Deployment {
	labels := map[string]string{"app": "cloudMcp", "airbyte.io/component-role": "workload", "app.kubernetes.io/instance": release}
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "agents"}, Spec: appsv1.DeploymentSpec{Selector: &metav1.LabelSelector{MatchLabels: labels}}}
}

func TestCloudMCPCredentialCreateAndRotation(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset(cloudDeployment("cloud", "install"), cloudDeployment("other", "different-release"))
	kube := &DefaultK8sClient{ClientSet: client}
	lastRestart := ""
	for _, secret := range []string{"first-test-value", "second-test-value"} {
		require.NoError(t, kube.ConfigureCloudMCP(ctx, "agents", "install", "install-client", secret))
		stored, err := client.CoreV1().Secrets("agents").Get(ctx, "cloud-mcp-oidc", metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, "install-client", string(stored.Data["clientId"]))
		require.Equal(t, secret, string(stored.Data["clientSecret"]))
		deployment, err := client.AppsV1().Deployments("agents").Get(ctx, "cloud", metav1.GetOptions{})
		require.NoError(t, err)
		restarted := deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]
		require.NotEmpty(t, restarted)
		require.NotEqual(t, lastRestart, restarted)
		lastRestart = restarted
	}
	actions := client.Actions()
	wroteSecret := false
	patches := 0
	for _, action := range actions {
		if action.GetResource().Resource == "secrets" && (action.GetVerb() == "create" || action.GetVerb() == "update") {
			wroteSecret = true
		}
		if action.GetResource().Resource == "deployments" && action.GetVerb() == "patch" {
			require.True(t, wroteSecret)
			require.Equal(t, "cloud", action.(ktesting.PatchAction).GetName())
			patches++
		}
	}
	require.Equal(t, 2, patches)
}

func TestCloudMCPCredentialConflictPreservesMetadata(t *testing.T) {
	client := fake.NewSimpleClientset(cloudDeployment("cloud", "install"), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "cloud-mcp-oidc", Namespace: "agents", ResourceVersion: "7", Labels: map[string]string{"keep": "yes"}}, Data: map[string][]byte{"unrelated": []byte("retained")}})
	updates := 0
	client.PrependReactor("update", "secrets", func(action ktesting.Action) (bool, runtime.Object, error) {
		updates++
		if updates == 1 {
			return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "secrets"}, "cloud-mcp-oidc", errors.New("conflict"))
		}
		return false, nil, nil
	})
	require.NoError(t, (&DefaultK8sClient{ClientSet: client}).ConfigureCloudMCP(context.Background(), "agents", "install", "id", "test-only"))
	require.Equal(t, 2, updates)
	secret, err := client.CoreV1().Secrets("agents").Get(context.Background(), "cloud-mcp-oidc", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "7", secret.ResourceVersion)
	require.Equal(t, "yes", secret.Labels["keep"])
	require.Equal(t, "retained", string(secret.Data["unrelated"]))
}

func TestCloudMCPCredentialErrors(t *testing.T) {
	for _, tc := range []struct{ name, verb, resource, expected string }{
		{"secret read", "get", "secrets", "Secret write failed"},
		{"secret create", "create", "secrets", "Secret write failed"},
		{"deployment list", "list", "deployments", "deployment lookup failed"},
		{"restart", "patch", "deployments", "deployment restart failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(cloudDeployment("cloud", "install"))
			client.PrependReactor(tc.verb, tc.resource, func(ktesting.Action) (bool, runtime.Object, error) { return true, nil, errors.New("forbidden") })
			err := (&DefaultK8sClient{ClientSet: client}).ConfigureCloudMCP(context.Background(), "agents", "install", "id", "test-only")
			require.ErrorContains(t, err, tc.expected)
			if tc.resource == "secrets" {
				for _, action := range client.Actions() {
					require.NotEqual(t, "patch", action.GetVerb())
				}
			}
		})
	}
	for _, count := range []int{0, 2} {
		client := fake.NewSimpleClientset()
		for i := 0; i < count; i++ {
			_, err := client.AppsV1().Deployments("agents").Create(context.Background(), cloudDeployment(string(rune('a'+i)), "install"), metav1.CreateOptions{})
			require.NoError(t, err)
		}
		require.ErrorContains(t, (&DefaultK8sClient{ClientSet: client}).ConfigureCloudMCP(context.Background(), "agents", "install", "id", "test-only"), "exactly one deployment")
	}
	client := fake.NewSimpleClientset()
	require.ErrorContains(t, (&DefaultK8sClient{ClientSet: client}).ConfigureCloudMCP(context.Background(), "agents", "install", "id", ""), "incomplete credentials")
	require.Empty(t, client.Actions())
	t.Setenv("KUBECONFIG", t.TempDir()+"/missing")
	_, err := NewComponentClient("", "unknown-context")
	require.Error(t, err)
}
