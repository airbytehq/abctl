package local

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s/k8stest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pterm/pterm"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploymentsCmd(t *testing.T) {
	b := bytes.NewBufferString("")
	pterm.SetDefaultOutput(b)
	pterm.EnableDebugMessages()
	// remove color codes from output
	pterm.DisableColor()
	t.Cleanup(func() {
		pterm.SetDefaultOutput(os.Stdout)
		pterm.DisableDebugMessages()
		pterm.EnableColor()
	})

	ctx := context.Background()
	t.Run("two deployments", func(t *testing.T) {
		b.Reset()

		mockK8s := &k8stest.MockClient{
			FnDeploymentList: func(ctx context.Context, namespace string) (*v1.DeploymentList, error) {
				if d := cmp.Diff(airbyteNamespace, namespace); d != "" {
					t.Errorf("unexpected namespace:\n%s", d)
				}

				return &v1.DeploymentList{
					Items: []v1.Deployment{
						{ObjectMeta: metav1.ObjectMeta{Name: "deployment0"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "deployment1"}},
					},
				}, nil
			},
		}

		cmd := &DeploymentsCmd{}
		err := cmd.deployments(ctx, mockK8s, &pterm.DefaultSpinner)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !strings.Contains(b.String(), "deployment0") {
			t.Error("missing deployment0 from output")
		}
		if !strings.Contains(b.String(), "deployment1") {
			t.Error("missing deployment1 from output")
		}
	})

	t.Run("no deployments", func(t *testing.T) {
		b.Reset()

		mockK8s := &k8stest.MockClient{
			FnDeploymentList: func(ctx context.Context, namespace string) (*v1.DeploymentList, error) {
				if d := cmp.Diff(airbyteNamespace, namespace); d != "" {
					t.Errorf("unexpected namespace:\n%s", d)
				}

				return &v1.DeploymentList{Items: []v1.Deployment{}}, nil
			},
		}

		cmd := &DeploymentsCmd{}
		err := cmd.deployments(ctx, mockK8s, &pterm.DefaultSpinner)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !strings.Contains(b.String(), "No deployments found") {
			t.Error("missing 'No deployments found' from output")
		}
	})

	t.Run("error", func(t *testing.T) {
		b.Reset()

		errTest := errors.New("test error")
		mockK8s := &k8stest.MockClient{
			FnDeploymentList: func(ctx context.Context, namespace string) (*v1.DeploymentList, error) {
				return nil, errTest
			},
		}

		cmd := &DeploymentsCmd{}
		err := cmd.deployments(ctx, mockK8s, &pterm.DefaultSpinner)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("error mismatch (-want +got):\n%s", d)
		}
	})
}
