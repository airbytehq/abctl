package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/airbytehq/abctl/internal/common"
	containerruntime "github.com/airbytehq/abctl/internal/container"
	"github.com/airbytehq/abctl/internal/paths"
	"github.com/airbytehq/abctl/internal/trace"
	"github.com/pterm/pterm"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/log"
)

// Provider represents a k8s provider.
type Provider struct {
	// Name of this provider
	Name string
	// ClusterName is the name of the cluster this provider will interact with
	ClusterName string
	// Context this provider should use
	Context string
	// Kubeconfig location
	Kubeconfig string
	// ContainerRuntime specifies which container runtime to use (optional)
	ContainerRuntime containerruntime.Runtime
}

// Cluster returns a kubernetes cluster for this provider.
func (p Provider) Cluster(ctx context.Context) (Cluster, error) {
	_, span := trace.NewSpan(ctx, "Provider.Cluster")
	defer span.End()

	if err := os.MkdirAll(filepath.Dir(p.Kubeconfig), 0o766); err != nil {
		return nil, fmt.Errorf("unable to create directory %s: %v", p.Kubeconfig, err)
	}

	// Set up KIND provider options based on container runtime
	providerOpts := []cluster.ProviderOption{
		cluster.ProviderWithLogger(&kindLogger{pterm: pterm.Debug}),
	}

	// Add container runtime option if specified
	switch p.ContainerRuntime {
	case containerruntime.Podman:
		pterm.Debug.Println("Using Podman provider for KIND cluster")
		providerOpts = append(providerOpts, cluster.ProviderWithPodman())
	case containerruntime.Docker:
		pterm.Debug.Println("Using Docker provider for KIND cluster")
		providerOpts = append(providerOpts, cluster.ProviderWithDocker())
	case containerruntime.Auto:
		// Let KIND auto-detect
		pterm.Debug.Println("Auto-detecting container runtime for KIND cluster")
		// Auto-detection is handled by checking environment variables
		// Set the appropriate environment variable if not already set
		if os.Getenv("KIND_EXPERIMENTAL_PROVIDER") == "" {
			// Try to detect what we should use
			if runtime, err := p.detectContainerRuntime(ctx); err == nil {
				switch runtime {
				case containerruntime.Podman:
					os.Setenv("KIND_EXPERIMENTAL_PROVIDER", "podman")
					providerOpts = append(providerOpts, cluster.ProviderWithPodman())
				case containerruntime.Docker:
					os.Setenv("KIND_EXPERIMENTAL_PROVIDER", "docker")
					providerOpts = append(providerOpts, cluster.ProviderWithDocker())
				}
			}
		}
	default:
		// Default behavior - let KIND choose
		pterm.Debug.Println("Using default container runtime detection for KIND cluster")
	}

	kindProvider := cluster.NewProvider(providerOpts...)
	if err := kindProvider.ExportKubeConfig(p.ClusterName, p.Kubeconfig, false); err != nil {
		pterm.Debug.Printfln("failed to export kube config: %s", err)
	}

	return &KindCluster{
		p:           kindProvider,
		kubeconfig:  p.Kubeconfig,
		clusterName: p.ClusterName,
	}, nil
}

// detectContainerRuntime detects the available container runtime
func (p Provider) detectContainerRuntime(ctx context.Context) (containerruntime.Runtime, error) {
	detector := containerruntime.NewAutoDetector()
	runtime, _, err := detector.DetectRuntime(ctx)
	return runtime, err
}

var (
	_ log.Logger     = (*kindLogger)(nil)
	_ log.InfoLogger = (*kindLogger)(nil)
)

// kindLogger implements the k8s logger interfaces.
// Necessarily in order to capture kind specify logging for debug purposes
type kindLogger struct {
	pterm pterm.PrefixPrinter
}

func (k *kindLogger) Info(message string) {
	k.pterm.Println("kind - INFO: " + message)
}

func (k *kindLogger) Infof(format string, args ...interface{}) {
	k.pterm.Println(fmt.Sprintf("kind - INFO: "+format, args...))
}

func (k *kindLogger) Enabled() bool {
	return true
}

func (k *kindLogger) Warn(message string) {
	k.pterm.Println("kind - WARN: " + message)
}

func (k *kindLogger) Warnf(format string, args ...interface{}) {
	k.pterm.Println(fmt.Sprintf("kind - WARN: "+format, args...))
}

func (k *kindLogger) Error(message string) {
	k.pterm.Println("kind - ERROR: " + message)
}

func (k *kindLogger) Errorf(format string, args ...interface{}) {
	k.pterm.Println(fmt.Sprintf("kind - ERROR: "+format, args...))
}

func (k *kindLogger) V(_ log.Level) log.InfoLogger {
	return k
}

const (
	Kind = "kind"
	Test = "test"
)

var (
	// DefaultProvider represents the kind (https://kind.sigs.k8s.io/) provider.
	DefaultProvider = Provider{
		Name:             Kind,
		ClusterName:      "airbyte-abctl",
		Context:          common.AirbyteKubeContext,
		Kubeconfig:       paths.Kubeconfig,
		ContainerRuntime: containerruntime.Auto, // Auto-detect container runtime
	}

	// TestProvider represents a test provider, for testing purposes
	TestProvider = Provider{
		Name:             Test,
		ClusterName:      "test-airbyte-abctl",
		Context:          "test-airbyte-abctl",
		Kubeconfig:       filepath.Join(os.TempDir(), "abctl", paths.FileKubeconfig),
		ContainerRuntime: containerruntime.Auto, // Auto-detect container runtime
	}
)

// WithContainerRuntime returns a provider configured with the specified container runtime
func WithContainerRuntime(runtime containerruntime.Runtime) Provider {
	provider := DefaultProvider
	provider.ContainerRuntime = runtime
	return provider
}

// WithPodman returns a provider configured to use Podman
func WithPodman() Provider {
	return WithContainerRuntime(containerruntime.Podman)
}

// WithDocker returns a provider configured to use Docker
func WithDocker() Provider {
	return WithContainerRuntime(containerruntime.Docker)
}
