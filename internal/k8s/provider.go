package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
}

// Cluster returns a kubernetes cluster for this provider.
func (p Provider) Cluster(ctx context.Context) (Cluster, error) {
	_, span := trace.NewSpan(ctx, "Provider.Cluster")
	defer span.End()

	if err := os.MkdirAll(filepath.Dir(p.Kubeconfig), 0766); err != nil {
		return nil, fmt.Errorf("unable to create directory %s: %v", p.Kubeconfig, err)
	}

	kindProvider := cluster.NewProvider(cluster.ProviderWithLogger(&kindLogger{pterm: pterm.Debug}))
	if err := kindProvider.ExportKubeConfig(p.ClusterName, p.Kubeconfig, false); err != nil {
		pterm.Debug.Printfln("failed to export kube config: %s", err)
	}

	return &kindCluster{
		p:           kindProvider,
		kubeconfig:  p.Kubeconfig,
		clusterName: p.ClusterName,
	}, nil
}

var _ log.Logger = (*kindLogger)(nil)
var _ log.InfoLogger = (*kindLogger)(nil)

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
		Name:        Kind,
		ClusterName: "airbyte-abctl",
		Context:     "kind-airbyte-abctl",
		Kubeconfig:  paths.Kubeconfig,
	}

	// TestProvider represents a test provider, for testing purposes
	TestProvider = Provider{
		Name:        Test,
		ClusterName: "test-airbyte-abctl",
		Context:     "test-airbyte-abctl",
		Kubeconfig:  filepath.Join(os.TempDir(), "abctl", paths.FileKubeconfig),
	}
)
