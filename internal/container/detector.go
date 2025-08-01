package container

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/airbytehq/abctl/internal/paths"
	"github.com/pterm/pterm"
)

// DockerSocketDetector detects Docker socket paths
type DockerSocketDetector struct {
	executor CommandExecutor
}

// NewDockerSocketDetector creates a new Docker socket detector
func NewDockerSocketDetector() *DockerSocketDetector {
	return &DockerSocketDetector{
		executor: NewDockerExecutor(),
	}
}

// DetectSockets detects available Docker socket paths
func (d *DockerSocketDetector) DetectSockets(ctx context.Context, goos string) ([]string, error) {
	var potentialHosts []string

	// Use the Docker CLI to get context information
	if contexts, err := GetDockerContexts(ctx, d.executor); err == nil && len(contexts) > 0 {
		if contexts[0].Endpoints.Docker.Host != "" {
			potentialHosts = append(potentialHosts, contexts[0].Endpoints.Docker.Host)
		}
	}

	// If the code above fails, then fall back to some educated guesses.
	// Unfortunately, these can easily be wrong if the user is using a non-standard
	// docker context, or if we've missed any common installation configs here.
	switch goos {
	case "darwin":
		potentialHosts = append(potentialHosts,
			"unix:///var/run/docker.sock",
			fmt.Sprintf("unix://%s/.docker/run/docker.sock", paths.UserHome),
		)
	case "windows":
		potentialHosts = append(potentialHosts, "npipe:////./pipe/docker_engine")
	default:
		potentialHosts = append(potentialHosts,
			"unix:///var/run/docker.sock",
			fmt.Sprintf("unix://%s/.docker/desktop/docker-cli.sock", paths.UserHome),
		)
	}

	return potentialHosts, nil
}

// PodmanSocketDetector detects Podman socket paths
type PodmanSocketDetector struct {
	executor CommandExecutor
}

// NewPodmanSocketDetector creates a new Podman socket detector
func NewPodmanSocketDetector() *PodmanSocketDetector {
	return &PodmanSocketDetector{
		executor: NewPodmanExecutor(),
	}
}

// DetectSockets detects available Podman socket paths
func (p *PodmanSocketDetector) DetectSockets(ctx context.Context, goos string) ([]string, error) {
	var potentialHosts []string

	// Try to get connection information from Podman CLI
	if connections, err := GetPodmanConnections(ctx, p.executor); err == nil {
		for _, conn := range connections {
			if conn.URI != "" {
				potentialHosts = append(potentialHosts, conn.URI)
			}
		}
	}

	// Check for rootless socket first (preferred for security)
	if uid := os.Getuid(); uid != 0 {
		xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
		if xdgRuntime == "" {
			xdgRuntime = fmt.Sprintf("/run/user/%d", uid)
		}
		potentialHosts = append(potentialHosts,
			fmt.Sprintf("unix://%s/podman/podman.sock", xdgRuntime))
	}

	// Check for rootful socket
	potentialHosts = append(potentialHosts, "unix:///run/podman/podman.sock")

	// Check for Docker compatibility socket (common symlink setup)
	potentialHosts = append(potentialHosts, "unix:///var/run/docker.sock")

	// Platform-specific paths
	switch goos {
	case "darwin":
		// Podman machine on macOS
		potentialHosts = append(potentialHosts,
			fmt.Sprintf("unix://%s/.local/share/containers/podman/machine/podman-machine-default/podman.sock", paths.UserHome),
		)
	case "windows":
		// Podman on Windows typically uses named pipes similar to Docker
		potentialHosts = append(potentialHosts, "npipe:////./pipe/podman_engine")
	}

	return potentialHosts, nil
}

// AutoDetector tries to detect the best available container runtime
type AutoDetector struct {
	dockerDetector *DockerSocketDetector
	podmanDetector *PodmanSocketDetector
}

// NewAutoDetector creates a new auto-detector
func NewAutoDetector() *AutoDetector {
	return &AutoDetector{
		dockerDetector: NewDockerSocketDetector(),
		podmanDetector: NewPodmanSocketDetector(),
	}
}

// DetectRuntime attempts to detect the available container runtime
func (a *AutoDetector) DetectRuntime(ctx context.Context) (Runtime, []string, error) {
	goos := runtime.GOOS

	pterm.Debug.Printfln("Starting runtime detection on %s", goos)

	// Try Podman first (preferred for rootless environments)
	pterm.Debug.Println("Trying Podman detection...")
	if podmanSockets, err := a.podmanDetector.DetectSockets(ctx, goos); err == nil && len(podmanSockets) > 0 {
		pterm.Debug.Printfln("Found Podman sockets: %v", podmanSockets)
		// Check if any Podman socket is actually available
		if a.isRuntimeAvailable(ctx, podmanSockets) {
			pterm.Debug.Println("Podman runtime available")
			return Podman, podmanSockets, nil
		} else {
			pterm.Debug.Println("Podman sockets found but not available")
		}
	} else {
		pterm.Debug.Printfln("Podman socket detection failed: %v", err)
	}

	// Fall back to Docker
	pterm.Debug.Println("Trying Docker detection...")
	if dockerSockets, err := a.dockerDetector.DetectSockets(ctx, goos); err == nil && len(dockerSockets) > 0 {
		pterm.Debug.Printfln("Found Docker sockets: %v", dockerSockets)
		if a.isRuntimeAvailable(ctx, dockerSockets) {
			pterm.Debug.Println("Docker runtime available")
			return Docker, dockerSockets, nil
		} else {
			pterm.Debug.Println("Docker sockets found but not available")
		}
	} else {
		pterm.Debug.Printfln("Docker socket detection failed: %v", err)
	}

	return Auto, nil, fmt.Errorf("no container runtime detected")
}

// isRuntimeAvailable checks if any of the provided sockets are accessible
func (a *AutoDetector) isRuntimeAvailable(ctx context.Context, sockets []string) bool {
	for _, socket := range sockets {
		// Simple check: see if the socket path exists for unix sockets
		if len(socket) > 7 && socket[:7] == "unix://" {
			socketPath := socket[7:]
			if stat, err := os.Stat(socketPath); err == nil {
				// Check if it's actually a socket
				if stat.Mode()&os.ModeSocket != 0 {
					pterm.Debug.Printfln("Found available socket: %s", socket)
					return true
				}
			} else {
				pterm.Debug.Printfln("Socket not accessible: %s (%v)", socket, err)
			}
		}
		// For other protocols (npipe, tcp), we'd need to try connecting
		// For now, assume they're available if listed
		if len(socket) > 7 && socket[:7] != "unix://" {
			pterm.Debug.Printfln("Assuming non-unix socket is available: %s", socket)
			return true
		}
	}
	pterm.Debug.Printfln("No available sockets found in: %v", sockets)
	return false
}