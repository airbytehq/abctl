package cmd

import (
	"context"

	"github.com/airbytehq/abctl/internal/cmd/images"
	"github.com/airbytehq/abctl/internal/cmd/local"
	"github.com/airbytehq/abctl/internal/cmd/version"
	containerruntime "github.com/airbytehq/abctl/internal/container"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
)

type verbose bool

func (v verbose) BeforeApply() error {
	pterm.EnableDebugMessages()
	return nil
}

type Cmd struct {
	Local   local.Cmd   `cmd:"" help:"Manage the local Airbyte installation."`
	Images  images.Cmd  `cmd:"" help:"Manage images used by Airbyte and abctl."`
	Version version.Cmd `cmd:"" help:"Display version information."`
	Verbose verbose     `short:"v" help:"Enable verbose output."`
}

func (c *Cmd) BeforeApply(ctx context.Context, kCtx *kong.Context) error {
	// Determine the container runtime and create appropriate provider
	config := containerruntime.LoadConfig()
	
	var provider k8s.Provider
	switch config.Runtime {
	case containerruntime.Podman:
		pterm.Debug.Println("Using Podman provider for Kubernetes cluster")
		provider = k8s.WithPodman()
	case containerruntime.Docker:
		pterm.Debug.Println("Using Docker provider for Kubernetes cluster")
		provider = k8s.WithDocker()
	case containerruntime.Auto:
		pterm.Debug.Println("Auto-detecting container runtime for Kubernetes cluster")
		// Try to auto-detect
		if detector := containerruntime.NewAutoDetector(); detector != nil {
			if runtime, _, err := detector.DetectRuntime(ctx); err == nil {
				switch runtime {
				case containerruntime.Podman:
					pterm.Debug.Println("Auto-detected Podman, using Podman provider")
					provider = k8s.WithPodman()
				case containerruntime.Docker:
					pterm.Debug.Println("Auto-detected Docker, using Docker provider")
					provider = k8s.WithDocker()
				default:
					pterm.Debug.Println("Using default provider with auto-detection")
					provider = k8s.DefaultProvider
				}
			} else {
				pterm.Debug.Printfln("Auto-detection failed, using default provider: %v", err)
				provider = k8s.DefaultProvider
			}
		} else {
			provider = k8s.DefaultProvider
		}
	default:
		pterm.Debug.Println("Using default Kubernetes provider")
		provider = k8s.DefaultProvider
	}
	
	kCtx.BindTo(provider, (*k8s.Provider)(nil))
	kCtx.BindTo(service.DefaultManagerClientFactory, (*service.ManagerClientFactory)(nil))
	return nil
}
