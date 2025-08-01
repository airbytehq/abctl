package container

import (
	"context"
	"fmt"
	"runtime"

	"github.com/airbytehq/abctl/internal/abctl"
	"github.com/docker/docker/client"
	"github.com/pterm/pterm"
	"go.opentelemetry.io/otel/sdk/trace"
)

// DefaultClientFactory is the default implementation of ClientFactory
type DefaultClientFactory struct {
	config   *Config
	detector *AutoDetector
}

// NewClientFactory creates a new client factory with the given configuration
func NewClientFactory(config *Config) *DefaultClientFactory {
	if config == nil {
		config = LoadConfig()
	}
	return &DefaultClientFactory{
		config:   config,
		detector: NewAutoDetector(),
	}
}

// CreateClient creates a container client for the specified runtime
func (f *DefaultClientFactory) CreateClient(ctx context.Context, runtime Runtime) (Client, error) {
	switch runtime {
	case Docker:
		return f.createDockerClient(ctx)
	case Podman:
		return f.createPodmanClient(ctx)
	case Auto:
		return f.autoDetectClient(ctx)
	default:
		return nil, fmt.Errorf("unsupported runtime: %v", runtime)
	}
}

// autoDetectClient automatically detects and creates the best available client
func (f *DefaultClientFactory) autoDetectClient(ctx context.Context) (Client, error) {
	detectedRuntime, sockets, err := f.detector.DetectRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to detect container runtime: %w", err)
	}

	switch detectedRuntime {
	case Podman:
		pterm.Debug.Println("Auto-detected Podman runtime")
		return f.createPodmanClient(ctx)
	case Docker:
		pterm.Debug.Println("Auto-detected Docker runtime")
		return f.createDockerClient(ctx)
	default:
		return nil, fmt.Errorf("no suitable container runtime found in sockets: %v", sockets)
	}
}

// createDockerClient creates a Docker client
func (f *DefaultClientFactory) createDockerClient(ctx context.Context) (Client, error) {
	detector := NewDockerSocketDetector()
	return f.createClientWithDetector(ctx, detector, "Docker")
}

// createPodmanClient creates a Podman client using Docker API compatibility
func (f *DefaultClientFactory) createPodmanClient(ctx context.Context) (Client, error) {
	detector := NewPodmanSocketDetector()
	return f.createClientWithDetector(ctx, detector, "Podman")
}

// createClientWithDetector creates a client using the specified socket detector
func (f *DefaultClientFactory) createClientWithDetector(ctx context.Context, detector SocketDetector, runtimeName string) (Client, error) {
	var potentialHosts []string
	var err error

	// Use explicit socket path if configured
	if f.config.SocketPath != "" {
		potentialHosts = []string{f.config.SocketPath}
	} else {
		potentialHosts, err = detector.DetectSockets(ctx, runtime.GOOS)
		if err != nil {
			return nil, fmt.Errorf("unable to detect %s sockets: %w", runtimeName, err)
		}
	}

	// Do not sample container runtime traces. Net/HTTP client has Otel instrumentation enabled.
	// URLs and other fields may contain PII, or sensitive information.
	noopTraceProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.NeverSample()),
	)

	dockerOpts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithTraceProvider(noopTraceProvider),
	}

	for _, host := range potentialHosts {
		dockerCli, err := f.createAndPing(ctx, host, dockerOpts, runtimeName)
		if err != nil {
			pterm.Debug.Printfln("error connecting to %s host %s: %s", runtimeName, host, err)
		} else {
			return &RuntimeClient{
				client:      dockerCli,
				runtimeType: runtimeName,
			}, nil
		}
	}

	return nil, fmt.Errorf("%w: unable to create %s client", abctl.ErrDocker, runtimeName)
}

// createAndPing attempts to create a client and ping it to ensure we can communicate
func (f *DefaultClientFactory) createAndPing(ctx context.Context, host string, opts []client.Opt, runtimeName string) (*client.Client, error) {
	// Pass client.WithHost first to ensure it runs prior to the client.FromEnv call.
	// We want the DOCKER_HOST to be used if it has been specified, overriding our host.
	cli, err := client.NewClientWithOpts(append([]client.Opt{client.WithHost(host)}, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s client: %w", runtimeName, err)
	}

	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("unable to ping %s client: %w", runtimeName, err)
	}

	return cli, nil
}

// New creates a new ContainerRuntime with auto-detection
func New(ctx context.Context) (*ContainerRuntime, error) {
	return NewWithConfig(ctx, nil)
}

// NewWithConfig creates a new ContainerRuntime with the specified configuration
func NewWithConfig(ctx context.Context, config *Config) (*ContainerRuntime, error) {
	if config == nil {
		config = LoadConfig()
	}

	// Create provider which includes both client and executor
	provider, err := NewProvider(ctx, config.Runtime)
	if err != nil {
		return nil, err
	}

	return &ContainerRuntime{
		Client:   provider.Client,
		Type:     provider.Runtime,
		config:   config,
		provider: provider,
	}, nil
}