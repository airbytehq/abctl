package container

import (
	"context"
	"fmt"
	"log"
)

// ExampleUsage demonstrates how to use the container runtime abstraction
// with both Docker and Podman support including CLI commands
func ExampleUsage() {
	ctx := context.Background()

	// Method 1: Auto-detect the available runtime
	fmt.Println("=== Auto-detection ===")
	runtime, err := New(ctx)
	if err != nil {
		log.Printf("Failed to create runtime: %v", err)
		return
	}

	version, err := runtime.Version(ctx)
	if err != nil {
		log.Printf("Failed to get version: %v", err)
	} else {
		fmt.Printf("Runtime: %s %s (%s/%s)\n", version.Runtime, version.Version, version.Platform, version.Arch)
	}

	fmt.Printf("Is rootless: %t\n", runtime.IsRootless())

	// Method 2: Explicitly use Podman
	fmt.Println("\n=== Explicit Podman ===")
	podmanConfig := &Config{
		Runtime:        Podman,
		PreferRootless: true,
	}
	
	podmanRuntime, err := NewWithConfig(ctx, podmanConfig)
	if err != nil {
		log.Printf("Failed to create Podman runtime: %v", err)
	} else {
		version, err := podmanRuntime.Version(ctx)
		if err != nil {
			log.Printf("Failed to get Podman version: %v", err)
		} else {
			fmt.Printf("Podman Runtime: %s %s\n", version.Runtime, version.Version)
		}

		// Demonstrate CLI executor
		if executor := podmanRuntime.Executor(); executor != nil {
			fmt.Printf("CLI executor: %s\n", executor.RuntimeName())
			
			// Get system info using CLI
			if capabilities, err := podmanRuntime.GetCapabilities(ctx); err == nil {
				fmt.Printf("Capabilities: %v\n", capabilities.GetRuntimeFeatures())
			}
		}
	}

	// Method 3: Use environment variable (like KIND)
	fmt.Println("\n=== Environment Variable Support ===")
	fmt.Println("Set KIND_EXPERIMENTAL_PROVIDER=podman or ABCTL_CONTAINER_RUNTIME=podman")
	fmt.Println("Then the runtime will automatically use Podman")

	// Method 4: Direct CLI executor usage
	fmt.Println("\n=== Direct CLI Usage ===")
	executor, err := AutoDetectExecutor(ctx)
	if err != nil {
		log.Printf("Failed to detect executor: %v", err)
	} else {
		fmt.Printf("Detected CLI executor: %s\n", executor.RuntimeName())
		
		// Execute a command directly
		if info, err := GetRuntimeInfo(ctx, executor); err == nil {
			fmt.Printf("Architecture: %s, CPUs: %d\n", info.Architecture, info.CPUs)
		}
		
		// Check if rootless
		if rootless, err := IsRootless(ctx, executor); err == nil {
			fmt.Printf("CLI detected rootless: %t\n", rootless)
		}
	}
}

// ExampleKindLikeUsage shows how to use the runtime similar to how KIND does it
func ExampleKindLikeUsage() {
	ctx := context.Background()

	// This mimics KIND's provider selection approach
	provider, err := GetDefault(ctx)
	if err != nil {
		log.Printf("Failed to get default provider: %v", err)
		return
	}

	fmt.Printf("Using provider: %s\n", provider.String())
	
	// Get system info like KIND does with "podman info --format json"
	sysInfo, err := provider.GetSystemInfo(ctx)
	if err != nil {
		log.Printf("Failed to get system info: %v", err)
	} else {
		fmt.Printf("System: %s, Cgroup: %s\n", sysInfo.OSType, sysInfo.CgroupVersion)
	}

	// Check rootless like KIND does
	rootless, err := provider.IsRootless(ctx)
	if err != nil {
		log.Printf("Failed to check rootless: %v", err)
	} else {
		fmt.Printf("Running rootless: %t\n", rootless)
	}

	// Execute raw CLI commands like KIND does
	if provider.Executor != nil {
		output, err := provider.Executor.Execute(ctx, "version", "--format", "json")
		if err != nil {
			log.Printf("Failed to execute version command: %v", err)
		} else {
			fmt.Printf("Version output: %s\n", string(output)[:min(100, len(output))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}