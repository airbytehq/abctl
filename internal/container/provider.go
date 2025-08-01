package container

import (
	"context"
	"fmt"
	"os"
)

// Provider represents a container runtime provider that can execute both
// API calls and CLI commands
type Provider struct {
	Runtime  Runtime
	Client   Client
	Executor CommandExecutor
	Config   *Config
}

// NewProvider creates a new container provider with the specified runtime
func NewProvider(ctx context.Context, runtime Runtime) (*Provider, error) {
	config := LoadConfig()
	if runtime == Auto {
		runtime = config.Runtime
	}

	factory := NewClientFactory(config)
	client, err := factory.CreateProvider(ctx, runtime)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	var executor CommandExecutor
	switch runtime {
	case Docker:
		executor = NewDockerExecutor()
	case Podman:
		executor = NewPodmanExecutor()
	case Auto:
		executor, err = AutoDetectExecutor(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-detect executor: %w", err)
		}
		// Update runtime based on detected executor
		if executor.RuntimeName() == "docker" {
			runtime = Docker
		} else if executor.RuntimeName() == "podman" {
			runtime = Podman
		}
	default:
		return nil, fmt.Errorf("unsupported runtime: %v", runtime)
	}

	return &Provider{
		Runtime:  runtime,
		Client:   client,
		Executor: executor,
		Config:   config,
	}, nil
}

// CreateProvider creates a client for the specified runtime (for backward compatibility)
func (f *DefaultClientFactory) CreateProvider(ctx context.Context, runtime Runtime) (Client, error) {
	return f.CreateClient(ctx, runtime)
}

// GetDefault returns the default provider based on environment configuration
func GetDefault(ctx context.Context) (*Provider, error) {
	config := LoadConfig()
	
	// Check explicit environment variable first
	if envRuntime := os.Getenv("ABCTL_CONTAINER_RUNTIME"); envRuntime != "" {
		return NewProvider(ctx, config.Runtime)
	}
	
	// Check KIND compatibility environment variable
	if envRuntime := os.Getenv("KIND_EXPERIMENTAL_PROVIDER"); envRuntime != "" {
		return NewProvider(ctx, config.Runtime)
	}
	
	// Auto-detect
	return NewProvider(ctx, Auto)
}

// IsRootless returns whether the provider is running in rootless mode
func (p *Provider) IsRootless(ctx context.Context) (bool, error) {
	return IsRootless(ctx, p.Executor)
}

// GetSystemInfo returns detailed system information from the runtime
func (p *Provider) GetSystemInfo(ctx context.Context) (*RuntimeInfo, error) {
	return GetRuntimeInfo(ctx, p.Executor)
}

// String returns a string representation of the provider
func (p *Provider) String() string {
	return fmt.Sprintf("%s provider", p.Runtime.String())
}