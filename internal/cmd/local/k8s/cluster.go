package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"sigs.k8s.io/kind/pkg/cluster"
	"time"
)

// Cluster is an interface representing all the actions taken at the cluster level.
type Cluster interface {
	// Create a cluster with the provided name.
	Create(portHTTP int) error
	// Delete a cluster with the provided name.
	Delete() error
	// Exists returns true if the cluster exists, false otherwise.
	Exists() bool
}

// interface sanity check
var _ Cluster = (*kindCluster)(nil)

// kindCluster is a Cluster implementation for kind (https://kind.sigs.k8s.io/).
type kindCluster struct {
	// p is the kind provider, not the abctl provider
	p *cluster.Provider
	// kubeconfig is the full path to the kubeconfig file kind is using
	kubeconfig  string
	clusterName string
}

const k8sVersion = "v1.29.1"

var mountPath = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".airbyte", "abctl", "data")
}

func (k *kindCluster) Create(port int) error {
	// see https://kind.sigs.k8s.io/docs/user/ingress/#create-cluster
	rawCfg := fmt.Sprintf(`kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
    - |
      kind: InitConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          node-labels: "ingress-ready=true"
    extraMounts:
      - hostPath: %s
        containerPath: /var/local-path-provisioner
    extraPortMappings:
      - containerPort: 80
        hostPort: %d
        protocol: TCP`,
		mountPath(),
		port)

	opts := []cluster.CreateOption{
		cluster.CreateWithWaitForReady(120 * time.Second),
		cluster.CreateWithKubeconfigPath(k.kubeconfig),
		cluster.CreateWithNodeImage("kindest/node:" + k8sVersion),
		cluster.CreateWithRawConfig([]byte(rawCfg)),
	}

	if err := k.p.Create(k.clusterName, opts...); err != nil {
		return fmt.Errorf("unable to create kind cluster: %w", err)
	}

	return nil
}

func (k *kindCluster) Delete() error {
	if err := k.p.Delete(k.clusterName, k.kubeconfig); err != nil {
		return fmt.Errorf("unable to delete kind cluster: %w", err)
	}

	return nil
}

func (k *kindCluster) Exists() bool {
	clusters, _ := k.p.List()
	for _, c := range clusters {
		if c == k.clusterName {
			return true
		}
	}

	return false
}

//func pvc(name string) *corev1.PersistentVolumeClaim {
//	size, _ := resource.ParseQuantity("500Mi")
//
//	return &corev1.PersistentVolumeClaim{
//		ObjectMeta: metav1.ObjectMeta{Name: name},
//		Spec: corev1.PersistentVolumeClaimSpec{
//			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
//			Resources: corev1.VolumeResourceRequirements{
//				Requests: corev1.ResourceList{corev1.ResourceStorage: size},
//			},
//			VolumeName:                "",
//			StorageClassName:          nil,
//			VolumeMode:                nil,
//			DataSource:                nil,
//			DataSourceRef:             nil,
//			VolumeAttributesClassName: nil,
//		},
//		Status: corev1.PersistentVolumeClaimStatus{},
//	}
//}

//func pv(name string) *corev1.PersistentVolume {
//	size, _ := resource.ParseQuantity("500Mi")
//
//	return &corev1.PersistentVolume{
//		ObjectMeta: metav1.ObjectMeta{Name: name},
//		Spec: corev1.PersistentVolumeSpec{
//			Capacity:               corev1.ResourceList{corev1.ResourceStorage: size},
//			PersistentVolumeSource: corev1.PersistentVolumeSource{},
//			AccessModes: []corev1.PersistentVolumeAccessMode{
//				corev1.ReadWriteOnce,
//			},
//			ClaimRef:                      nil,
//			PersistentVolumeReclaimPolicy: "",
//			StorageClassName:              "",
//			MountOptions:                  nil,
//			VolumeMode:                    nil,
//			NodeAffinity:                  nil,
//			VolumeAttributesClassName:     nil,
//		},
//	}
//}
