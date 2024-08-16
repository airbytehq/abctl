package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	errorsk8s "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apimachinery/pkg/watch"
	fake2 "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	testingk8s "k8s.io/client-go/testing"
)

const testNamespace = "test-namespace"

var (
	// errNotFound is returned to mimic a non-found error from the k8s client
	errNotFound = &errorsk8s.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
	errTest     = errors.New("test error")
)

func TestDefaultK8sClient_DeploymentRestart(t *testing.T) {
	testName := "deployment"

	deployment := &v1.Deployment{
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"abc": "xyz"},
			},
		},
	}

	now := time.Now()

	t.Run("happy path", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("patch", "deployments", func(action testingk8s.Action) (bool, runtime.Object, error) {
			patchAction, ok := action.(testingk8s.PatchAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}

			if d := cmp.Diff(testNamespace, patchAction.GetNamespace()); d != "" {
				return true, nil, fmt.Errorf("unexpected create namespace: %s", d)
			}
			if d := cmp.Diff(testName, patchAction.GetName()); d != "" {
				return true, nil, fmt.Errorf("unexpected create name: %s", d)
			}
			if d := cmp.Diff(types.StrategicMergePatchType, patchAction.GetPatchType()); d != "" {
				return true, nil, fmt.Errorf("unexpected patch type: %s", d)
			}

			var rawPatch map[string]any
			if err := json.Unmarshal(patchAction.GetPatch(), &rawPatch); err != nil {
				return true, nil, err
			}

			// here be dragons
			restartedAt := rawPatch["spec"].(map[string]any)["template"].(map[string]any)["metadata"].(map[string]any)["annotations"].(map[string]any)["kubectl.kubernetes.io/restartedAt"].(string)
			if d := cmp.Diff(restartedAt, now.Format(time.RFC3339)); d != "" {
				return true, nil, fmt.Errorf("unexpected restartedAt time: %s", d)
			}

			return true, deployment, nil
		})
		cs.PrependReactor("list", "pods", func(action testingk8s.Action) (handled bool, ret runtime.Object, err error) {
			listAction, ok := action.(testingk8s.ListAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}

			if d := cmp.Diff(testNamespace, listAction.GetNamespace()); d != "" {
				return true, nil, fmt.Errorf("unexpected create namespace: %s", d)
			}
			if d := cmp.Diff("abc=xyz", listAction.GetListRestrictions().Labels.String()); d != "" {
				return true, nil, fmt.Errorf("unexpected list labels: %s", d)
			}

			pods := &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"kubectl.kubernetes.io/restartedAt": now.Format(time.RFC3339),
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			}

			return true, pods, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.deploymentRestart(context.Background(), testNamespace, testName, now, 5*time.Second)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error applying patch", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("patch", "deployments", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.deploymentRestart(context.Background(), testNamespace, testName, now, 5*time.Second)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error: %s", d)
		}
	})

	t.Run("error listing pods", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("patch", "deployments", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, deployment, nil
		})
		cs.PrependReactor("list", "pods", func(action testingk8s.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.deploymentRestart(context.Background(), testNamespace, testName, now, 5*time.Second)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error: %s", d)
		}
	})

	t.Run("error pod is never running", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("patch", "deployments", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, deployment, nil
		})
		cs.PrependReactor("list", "pods", func(action testingk8s.Action) (handled bool, ret runtime.Object, err error) {
			pods := &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"kubectl.kubernetes.io/restartedAt": now.Format(time.RFC3339)},
							Labels:      map[string]string{"abc": "xyz"},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodPending,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			}

			return true, pods, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.deploymentRestart(context.Background(), testNamespace, testName, now, 5*time.Second)
		// should timeout after 5 seconds
		if d := cmp.Diff(context.DeadlineExceeded, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error: %s", d)
		}
	})

	t.Run("error pod is never ready", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("patch", "deployments", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, deployment, nil
		})
		cs.PrependReactor("list", "pods", func(action testingk8s.Action) (handled bool, ret runtime.Object, err error) {
			pods := &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"kubectl.kubernetes.io/restartedAt": now.Format(time.RFC3339)},
							Labels:      map[string]string{"abc": "xyz"},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionFalse,
								},
							},
						},
					},
				},
			}

			return true, pods, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.deploymentRestart(context.Background(), testNamespace, testName, now, 5*time.Second)
		// should timeout after 5 seconds
		if d := cmp.Diff(context.DeadlineExceeded, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error: %s", d)
		}
	})

	t.Run("error pod is missing restartedAt attribute", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("patch", "deployments", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, deployment, nil
		})
		cs.PrependReactor("list", "pods", func(action testingk8s.Action) (handled bool, ret runtime.Object, err error) {
			pods := &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"abc": "xyz"},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			}

			return true, pods, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.deploymentRestart(context.Background(), testNamespace, testName, now, 5*time.Second)
		// should timeout after 5 seconds
		if d := cmp.Diff(context.DeadlineExceeded, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error: %s", d)
		}
	})
}

func TestDefaultK8sClient_IngressCreate(t *testing.T) {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: testNamespace,
		},
	}

	t.Run("happy path", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("create", "ingresses", func(action testingk8s.Action) (bool, runtime.Object, error) {
			createAction, ok := action.(testingk8s.CreateAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}

			incoming, ok := createAction.GetObject().(*networkingv1.Ingress)
			if !ok {
				return true, nil, fmt.Errorf("unexpected object type: %T", createAction.GetObject())
			}
			if d := cmp.Diff(testNamespace, incoming.ObjectMeta.Namespace); d != "" {
				return true, nil, fmt.Errorf("unexpected create namespace: %s", d)
			}
			if d := cmp.Diff("test-ingress", incoming.ObjectMeta.Name); d != "" {
				return true, nil, fmt.Errorf("unexpected create name: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.IngressCreate(context.Background(), testNamespace, ingress)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("create", "ingresses", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.IngressCreate(context.Background(), testNamespace, ingress)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error: %s", d)
		}
	})
}

func TestDefaultK8sClient_IngressUpdate(t *testing.T) {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: testNamespace,
		},
	}

	t.Run("happy path", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("update", "ingresses", func(action testingk8s.Action) (bool, runtime.Object, error) {
			createAction, ok := action.(testingk8s.CreateAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}

			incoming, ok := createAction.GetObject().(*networkingv1.Ingress)
			if !ok {
				return true, nil, fmt.Errorf("unexpected object type: %T", createAction.GetObject())
			}
			if d := cmp.Diff(testNamespace, incoming.ObjectMeta.Namespace); d != "" {
				return true, nil, fmt.Errorf("unexpected create namespace: %s", d)
			}
			if d := cmp.Diff("test-ingress", incoming.ObjectMeta.Name); d != "" {
				return true, nil, fmt.Errorf("unexpected create name: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.IngressUpdate(context.Background(), testNamespace, ingress)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("update", "ingresses", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.IngressUpdate(context.Background(), testNamespace, ingress)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error: %s", d)
		}
	})
}

func TestDefaultK8sClient_IngressExists(t *testing.T) {
	testName := "ingress"

	t.Run("exists", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "ingresses", func(action testingk8s.Action) (bool, runtime.Object, error) {
			getAction, ok := action.(testingk8s.GetAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}
			if d := cmp.Diff(testNamespace, getAction.GetNamespace()); d != "" {
				return true, nil, fmt.Errorf("unexpected ingress namespace: %s", d)
			}
			if d := cmp.Diff(testName, getAction.GetName()); d != "" {
				return true, nil, fmt.Errorf("unexpected ingress name: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.IngressExists(context.Background(), testNamespace, testName)
		if d := cmp.Diff(true, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})

	t.Run("does not exist", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "ingresses", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errNotFound
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.IngressExists(context.Background(), testNamespace, testName)
		if d := cmp.Diff(false, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "ingresses", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.IngressExists(context.Background(), testNamespace, testName)
		if d := cmp.Diff(true, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})
}

func TestDefaultK8sClient_NamespaceCreate(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("create", "namespaces", func(action testingk8s.Action) (bool, runtime.Object, error) {
			createAction, ok := action.(testingk8s.CreateAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}

			incoming, ok := createAction.GetObject().(*corev1.Namespace)
			if !ok {
				return true, nil, fmt.Errorf("unexpected object type: %T", createAction.GetObject())
			}
			if d := cmp.Diff(testNamespace, incoming.ObjectMeta.Name); d != "" {
				return true, nil, fmt.Errorf("unexpected create namespace: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.NamespaceCreate(context.Background(), testNamespace)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("create", "namespaces", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.NamespaceCreate(context.Background(), testNamespace)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error: %s", d)
		}
	})
}

func TestDefaultK8sClient_NamespaceExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "namespaces", func(action testingk8s.Action) (bool, runtime.Object, error) {
			getAction, ok := action.(testingk8s.GetAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}
			if d := cmp.Diff("", getAction.GetNamespace()); d != "" {
				return true, nil, fmt.Errorf("unexpected delete namespace: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.NamespaceExists(context.Background(), testNamespace)
		if d := cmp.Diff(true, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})

	t.Run("does not exist", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "namespaces", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errNotFound
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.NamespaceExists(context.Background(), testNamespace)
		if d := cmp.Diff(false, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "namespaces", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.NamespaceExists(context.Background(), testNamespace)
		if d := cmp.Diff(true, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})
}

func TestDefaultK8sClient_NamespaceDelete(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("delete", "namespaces", func(action testingk8s.Action) (bool, runtime.Object, error) {
			deleteAction, ok := action.(testingk8s.DeleteAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}
			// there isn't a namespace on this call, should there be?
			if d := cmp.Diff("", deleteAction.GetNamespace()); d != "" {
				return true, nil, fmt.Errorf("unexpected delete namespace: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.NamespaceDelete(context.Background(), testNamespace)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("delete", "namespaces", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.NamespaceDelete(context.Background(), testNamespace)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Fatalf("unexpected error: %v", d)
		}
	})
}

func TestDefaultK8sClient_PersistentVolumeCreate(t *testing.T) {
	testName := "pvc"

	t.Run("happy path", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("create", "persistentvolumes", func(action testingk8s.Action) (bool, runtime.Object, error) {
			createAction, ok := action.(testingk8s.CreateAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}

			incoming, ok := createAction.GetObject().(*corev1.PersistentVolume)
			if !ok {
				return true, nil, fmt.Errorf("unexpected object type: %T", createAction.GetObject())
			}
			if d := cmp.Diff(testNamespace, incoming.ObjectMeta.Namespace); d != "" {
				return true, nil, fmt.Errorf("unexpected create namespace: %s", d)
			}
			if d := cmp.Diff(testName, incoming.ObjectMeta.Name); d != "" {
				return true, nil, fmt.Errorf("unexpected create name: %s", d)
			}
			if d := cmp.Diff(DefaultPersistentVolumeSize, incoming.Spec.Capacity[corev1.ResourceStorage]); d != "" {
				return true, nil, fmt.Errorf("unexpected capcity: %s", d)
			}
			if d := cmp.Diff(path.Join("/var/local-path-provisioner", testName), incoming.Spec.PersistentVolumeSource.HostPath.Path); d != "" {
				return true, nil, fmt.Errorf("unexpected host path: %s", d)
			}
			if d := cmp.Diff(corev1.HostPathDirectoryOrCreate, *incoming.Spec.PersistentVolumeSource.HostPath.Type); d != "" {
				return true, nil, fmt.Errorf("unexpected host type: %s", d)
			}
			if d := cmp.Diff(corev1.ReadWriteOnce, incoming.Spec.AccessModes[0]); d != "" {
				return true, nil, fmt.Errorf("unexpected access mode: %s", d)
			}
			if d := cmp.Diff(corev1.PersistentVolumeReclaimRetain, incoming.Spec.PersistentVolumeReclaimPolicy); d != "" {
				return true, nil, fmt.Errorf("unexpected reclaim policy: %s", d)
			}
			if d := cmp.Diff("standard", incoming.Spec.StorageClassName); d != "" {
				return true, nil, fmt.Errorf("unexpected storage class name: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.PersistentVolumeCreate(context.Background(), testNamespace, testName)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("create", "persistentvolumes", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.PersistentVolumeCreate(context.Background(), testNamespace, testName)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error: %s", d)
		}
	})
}

func TestDefaultK8sClient_PersistentVolumeExists(t *testing.T) {
	testName := "pvc"
	t.Run("exists", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "persistentvolumes", func(action testingk8s.Action) (bool, runtime.Object, error) {
			getAction, ok := action.(testingk8s.GetAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}
			if d := cmp.Diff(testName, getAction.GetName()); d != "" {
				return true, nil, fmt.Errorf("unexpected delete name: %s", d)
			}
			if d := cmp.Diff("", getAction.GetNamespace()); d != "" {
				return true, nil, fmt.Errorf("unexpected delete namespace: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.PersistentVolumeExists(context.Background(), testNamespace, testName)
		if d := cmp.Diff(true, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})

	t.Run("does not exist", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "persistentvolumes", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errNotFound
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.PersistentVolumeExists(context.Background(), testNamespace, testName)
		if d := cmp.Diff(false, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "persistentvolumes", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.PersistentVolumeExists(context.Background(), testNamespace, testName)
		if d := cmp.Diff(true, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})
}

func TestDefaultK8sClient_PersistentVolumeDelete(t *testing.T) {
	testName := "pvc"
	t.Run("happy path", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("delete", "persistentvolumes", func(action testingk8s.Action) (bool, runtime.Object, error) {
			deleteAction, ok := action.(testingk8s.DeleteAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}
			if d := cmp.Diff(testName, deleteAction.GetName()); d != "" {
				return true, nil, fmt.Errorf("unexpected delete name: %s", d)
			}
			// there isn't a namespace on this call, should there be?
			if d := cmp.Diff("", deleteAction.GetNamespace()); d != "" {
				return true, nil, fmt.Errorf("unexpected delete namespace: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.PersistentVolumeDelete(context.Background(), testNamespace, testName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("delete", "persistentvolumes", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.PersistentVolumeDelete(context.Background(), testNamespace, testName)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Fatalf("unexpected error: %v", d)
		}
	})
}

func TestDefaultK8sClient_PersistentVolumeClaimCreate(t *testing.T) {
	testName := "pvc"
	testVolume := "volume"

	t.Run("happy path", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("create", "persistentvolumeclaims", func(action testingk8s.Action) (bool, runtime.Object, error) {
			createAction, ok := action.(testingk8s.CreateAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}

			incoming, ok := createAction.GetObject().(*corev1.PersistentVolumeClaim)
			if !ok {
				return true, nil, fmt.Errorf("unexpected object type: %T", createAction.GetObject())
			}
			if d := cmp.Diff(testNamespace, incoming.ObjectMeta.Namespace); d != "" {
				return true, nil, fmt.Errorf("unexpected create namespace: %s", d)
			}
			if d := cmp.Diff(testName, incoming.ObjectMeta.Name); d != "" {
				return true, nil, fmt.Errorf("unexpected create name: %s", d)
			}
			if d := cmp.Diff(corev1.ReadWriteOnce, incoming.Spec.AccessModes[0]); d != "" {
				return true, nil, fmt.Errorf("unexpected access mode: %s", d)
			}
			if d := cmp.Diff(DefaultPersistentVolumeSize, incoming.Spec.Resources.Requests[corev1.ResourceStorage]); d != "" {
				return true, nil, fmt.Errorf("unexpected resource storage: %s", d)
			}
			if d := cmp.Diff(testVolume, incoming.Spec.VolumeName); d != "" {
				return true, nil, fmt.Errorf("unexpected volume name: %s", d)
			}
			if d := cmp.Diff("standard", *incoming.Spec.StorageClassName); d != "" {
				return true, nil, fmt.Errorf("unexpected create storage class: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.PersistentVolumeClaimCreate(context.Background(), testNamespace, testName, testVolume)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("create", "persistentvolumeclaims", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.PersistentVolumeClaimCreate(context.Background(), testNamespace, testName, testVolume)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error: %s", d)
		}
	})
}

func TestDefaultK8sClient_PersistentVolumeClaimExists(t *testing.T) {
	testName := "pvc"
	t.Run("exists", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "persistentvolumeclaims", func(action testingk8s.Action) (bool, runtime.Object, error) {
			getAction, ok := action.(testingk8s.GetAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}
			if d := cmp.Diff("pvc", getAction.GetName()); d != "" {
				return true, nil, fmt.Errorf("unexpected delete name: %s", d)
			}
			if d := cmp.Diff(testNamespace, getAction.GetNamespace()); d != "" {
				return true, nil, fmt.Errorf("unexpected delete namespace: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.PersistentVolumeClaimExists(context.Background(), testNamespace, testName, "volume")
		if d := cmp.Diff(true, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})

	t.Run("does not exist", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "persistentvolumeclaims", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errNotFound
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.PersistentVolumeClaimExists(context.Background(), testNamespace, testName, "volume")
		if d := cmp.Diff(false, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "persistentvolumeclaims", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual := cli.PersistentVolumeClaimExists(context.Background(), testNamespace, testName, "volume")
		if d := cmp.Diff(true, actual); d != "" {
			t.Errorf("unexpected result: %s", d)
		}
	})
}

func TestDefaultK8sClient_PersistentVolumeClaimDelete(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("delete", "persistentvolumeclaims", func(action testingk8s.Action) (bool, runtime.Object, error) {
			deleteAction, ok := action.(testingk8s.DeleteAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}
			if d := cmp.Diff("pvc", deleteAction.GetName()); d != "" {
				return true, nil, fmt.Errorf("unexpected delete name: %s", d)
			}
			if d := cmp.Diff(testNamespace, deleteAction.GetNamespace()); d != "" {
				return true, nil, fmt.Errorf("unexpected delete namespace: %s", d)
			}

			return true, nil, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.PersistentVolumeClaimDelete(context.Background(), testNamespace, "pvc", "volume")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("delete", "persistentvolumeclaims", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.PersistentVolumeClaimDelete(context.Background(), testNamespace, "pvc", "volume")
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Fatalf("unexpected error: %v", d)
		}
	})
}

func TestDefaultK8sClient_SecretCreateOrUpdate(t *testing.T) {
	testSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: testNamespace,
		},
	}

	t.Run("no existing secret", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "secrets", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errNotFound
		})

		cs.PrependReactor("create", "secrets", func(action testingk8s.Action) (bool, runtime.Object, error) {
			createAction, ok := action.(testingk8s.CreateAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}
			if incoming, ok := createAction.GetObject().(*corev1.Secret); !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", createAction.GetObject())
			} else {
				if d := cmp.Diff(testSecret.ObjectMeta.Namespace, incoming.ObjectMeta.Namespace); d != "" {
					return true, nil, fmt.Errorf("unexpected namespace (-want +got):\n%s", d)
				}
				if d := cmp.Diff(testSecret.ObjectMeta.Name, incoming.ObjectMeta.Name); d != "" {
					return true, nil, fmt.Errorf("unexpected name (-want +got):\n%s", d)
				}
			}

			return true, &testSecret, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		if err := cli.SecretCreateOrUpdate(context.Background(), testSecret); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("no existing secret, fails to create", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "secrets", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errNotFound
		})

		cs.PrependReactor("create", "secrets", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.SecretCreateOrUpdate(context.Background(), testSecret)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Fatalf("unexpected error (-want +got):\n%s", d)
		}
	})

	t.Run("existing secret", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "secrets", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, &testSecret, nil
		})

		cs.PrependReactor("update", "secrets", func(action testingk8s.Action) (bool, runtime.Object, error) {
			updateAction, ok := action.(testingk8s.UpdateAction)
			if !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", action)
			}
			if incoming, ok := updateAction.GetObject().(*corev1.Secret); !ok {
				return true, nil, fmt.Errorf("unexpected action type: %T", updateAction.GetObject())
			} else {
				if d := cmp.Diff(testSecret.ObjectMeta.Namespace, incoming.ObjectMeta.Namespace); d != "" {
					return true, nil, fmt.Errorf("unexpected namespace (-want +got):\n%s", d)
				}
				if d := cmp.Diff(testSecret.ObjectMeta.Name, incoming.ObjectMeta.Name); d != "" {
					return true, nil, fmt.Errorf("unexpected name (-want +got):\n%s", d)
				}
			}

			return true, &testSecret, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		if err := cli.SecretCreateOrUpdate(context.Background(), testSecret); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("existing secret, fails to update", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "secrets", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, &testSecret, nil
		})

		cs.PrependReactor("update", "secrets", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		err := cli.SecretCreateOrUpdate(context.Background(), testSecret)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Fatalf("unexpected error (-want +got):\n%s", d)
		}
	})
}

func TestDefaultK8sClient_SecretGet(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		expected := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: testNamespace,
			},
		}

		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "secrets", func(action testingk8s.Action) (handled bool, ret runtime.Object, err error) {
			return true, expected, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual, err := cli.SecretGet(context.Background(), testNamespace, "test-secret")
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("Unexpected secret (-want, +got) = %v", diff)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "secrets", func(action testingk8s.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		_, err := cli.SecretGet(context.Background(), testNamespace, "test-secret")
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("Unexpected error (-want, +got) = %v", d)
		}
	})
}

func TestDefaultK8sClient_ServerVersionGet(t *testing.T) {
	expected := "v12.15"
	cs := fake.NewSimpleClientset()
	cs.Discovery().(*fake2.FakeDiscovery).FakedServerVersion = &version.Info{GitVersion: expected}

	cli := &DefaultK8sClient{ClientSet: cs}

	actual, err := cli.ServerVersionGet()
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(expected, actual); d != "" {
		t.Errorf("Unexpected server version (-want, +got): %s", d)
	}
}

func TestDefaultK8sClient_ServiceGet(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		expected := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-service",
				Namespace: testNamespace,
			},
		}
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "services", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, expected, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}

		actual, err := cli.ServiceGet(context.Background(), testNamespace, "test-service")
		if err != nil {
			t.Fatal(err)
		}
		if d := cmp.Diff(expected, actual); d != "" {
			t.Errorf("Unexpected service (-want, +got): %s", d)
		}
	})

	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependReactor("get", "services", func(action testingk8s.Action) (bool, runtime.Object, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}

		_, err := cli.ServiceGet(context.Background(), testNamespace, "test-service")
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("Unexpected error (-want, +got): %s", d)
		}
	})
}

func TestDefaultK8sClient_EventsWatch(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		expected := &watch.FakeWatcher{}
		cs := fake.NewSimpleClientset()
		cs.PrependWatchReactor("events", func(action testingk8s.Action) (bool, watch.Interface, error) {
			return true, expected, nil
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		actual, err := cli.EventsWatch(context.Background(), testNamespace)
		if err != nil {
			t.Fatal(err)
		}
		if d := cmp.Diff(expected, actual, cmpopts.EquateComparable(watch.FakeWatcher{})); d != "" {
			t.Errorf("Unexpected events (-want, +got): %s", d)
		}
	})
	t.Run("error", func(t *testing.T) {
		cs := fake.NewSimpleClientset()
		cs.PrependWatchReactor("events", func(action testingk8s.Action) (bool, watch.Interface, error) {
			return true, nil, errTest
		})

		cli := &DefaultK8sClient{ClientSet: cs}
		_, err := cli.EventsWatch(context.Background(), testNamespace)
		if d := cmp.Diff(errTest, err, cmpopts.EquateErrors()); d != "" {
			t.Errorf("Unexpected error (-want, +got): %s", d)
		}
	})
}

func TestDefaultK8sClient_LogsGet(t *testing.T) {
	ctx := context.Background()
	// the fake.ClientSet does not support custom logs, it always returns "fake logs"
	// see https://github.com/kubernetes/kubernetes/issues/125590
	cli := &DefaultK8sClient{ClientSet: fake.NewSimpleClientset()}
	logs, err := cli.LogsGet(ctx, testNamespace, "pod")
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff("fake logs", logs); d != "" {
		t.Errorf("Unexpected diff (-want, +got): %s", d)
	}
}
