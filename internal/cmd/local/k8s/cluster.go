package k8s

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/airbytehq/abctl/internal/cmd/local/k8s/kind"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/airbytehq/abctl/internal/trace"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	kindExec "sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
)

// ExtraVolumeMount defines a host volume mount for the Kind cluster
type ExtraVolumeMount struct {
	HostPath      string
	ContainerPath string
}

// Cluster is an interface representing all the actions taken at the cluster level.
type Cluster interface {
	// Create a cluster with the provided name.
	Create(ctx context.Context, portHTTP int, extraMounts []ExtraVolumeMount) error
	// Delete a cluster with the provided name.
	Delete(ctx context.Context) error
	// Exists returns true if the cluster exists, false otherwise.
	Exists(ctx context.Context) bool
	LoadImages(ctx context.Context, images []string)
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

// k8sVersion is the kind node version being used.
// Note that the sha256 must match the version listed on the release for the specific version of kind
// that we're currently using (e.g. https://github.com/kubernetes-sigs/kind/releases/tag/v0.24.0)
const k8sVersion = "v1.29.8@sha256:d46b7aa29567e93b27f7531d258c372e829d7224b25e3fc6ffdefed12476d3aa"

func (k *kindCluster) Create(ctx context.Context, port int, extraMounts []ExtraVolumeMount) error {
	ctx, span := trace.NewSpan(ctx, "kindCluster.Create")
	defer span.End()
	// Create the data directory before the cluster does to ensure that it's owned by the correct user.
	// If the cluster creates it and docker is running as root, it's possible that root will own this directory
	// which will cause minio and postgres to break.
	pterm.Debug.Println(fmt.Sprintf("Creating data directory '%s'", paths.Data))
	if err := os.MkdirAll(paths.Data, 0766); err != nil {
		pterm.Error.Println(fmt.Sprintf("Error creating data directory '%s'", paths.Data))
		return fmt.Errorf("unable to create directory '%s': %w", paths.Data, err)
	}

	// see https://kind.sigs.k8s.io/docs/user/ingress/#create-cluster
	config := kind.DefaultConfig().WithHostPort(port)
	for _, mount := range extraMounts {
		config = config.WithVolumeMount(mount.HostPath, mount.ContainerPath)
	}

	rawCfg, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("unable to marshal Kind cluster config: %w", err)
	}

	opts := []cluster.CreateOption{
		cluster.CreateWithWaitForReady(5 * time.Minute),
		cluster.CreateWithKubeconfigPath(k.kubeconfig),
		cluster.CreateWithNodeImage("kindest/node:" + k8sVersion),
		cluster.CreateWithRawConfig(rawCfg),
	}

	if err := k.p.Create(k.clusterName, opts...); err != nil {
		return fmt.Errorf("unable to create kind cluster: %w", formatKindErr(err))
	}

	return nil
}

func (k *kindCluster) Delete(ctx context.Context) error {
	_, span := trace.NewSpan(ctx, "kindCluster.Delete")
	defer span.End()
	if err := k.p.Delete(k.clusterName, k.kubeconfig); err != nil {
		return fmt.Errorf("unable to delete kind cluster: %w", formatKindErr(err))
	}

	return nil
}

func (k *kindCluster) Exists(ctx context.Context) bool {
	_, span := trace.NewSpan(ctx, "kindCluster.exists")
	defer span.End()

	clusters, _ := k.p.List()
	for _, c := range clusters {
		if c == k.clusterName {
			return true
		}
	}

	return false
}

// LoadImages pulls images from Docker Hub, and loads them into the kind cluster.
// This is a best-effort optimization, which is why it doesn't an error;
// it's possible that only some images will be loaded.
func (k *kindCluster) LoadImages(ctx context.Context, images []string) {
	err := k.loadImages(ctx, images)
	pterm.Debug.Printfln("failed to load images: %s", err)
}

func (k *kindCluster) loadImages(ctx context.Context, images []string) error {
	// Get a list of Kind nodes.
	nodes, err := k.p.ListNodes(k.clusterName)
	if err != nil {
		return fmt.Errorf("listing nodes: %w", err)
	}

	// Setup the tar path where the images will be saved.
	dir, err := fs.TempDir("", "images-tar-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	// Pull all the images via "docker pull", in parallel.
	var wg sync.WaitGroup
	wg.Add(len(images))
	for _, img := range images {
		pterm.Debug.Printfln("pulling image %s", img)

		go func(img string) {
			defer wg.Done()
			out, err := exec.CommandContext(ctx, "docker", "pull", img).CombinedOutput()
			if err != nil {
				pterm.Debug.Printfln("error pulling image %s", out)
				// don't return the error here, because other image pulls might succeed.
			}
		}(img)
	}
	wg.Wait()

	// The context could be canceled by now. If so, return early.
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Save all the images to an archive, images.tar
	imagesTarPath := filepath.Join(dir, "images.tar")
	pterm.Debug.Printfln("saving image archive to %s", imagesTarPath)

	out, err := exec.CommandContext(ctx, "docker", append([]string{"save", "-o", imagesTarPath}, images...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run 'docker save': %s", out)
	}
	
	// Load the image archive into the Kind nodes.
	f, err := os.Open(imagesTarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, n := range nodes {
		pterm.Debug.Printfln("loading image archive into kind node %s", n)
		nodeutils.LoadImageArchive(n, f)
	}
	return nil
}

func formatKindErr(err error) error {
	var kindErr *kindExec.RunError
	if errors.As(err, &kindErr) {
		return fmt.Errorf("%w: %s", err, string(kindErr.Output))
	}
	return err
}
