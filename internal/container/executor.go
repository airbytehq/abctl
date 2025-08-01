package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// CommandExecutor executes container runtime CLI commands
type CommandExecutor interface {
	// Execute runs a command with the given arguments
	Execute(ctx context.Context, args ...string) ([]byte, error)
	// RuntimeName returns the name of the runtime (docker, podman, etc.)
	RuntimeName() string
	// ContextInspect returns context information for the runtime
	ContextInspect(ctx context.Context) ([]byte, error)
	// Info returns runtime system information
	Info(ctx context.Context) ([]byte, error)
}

// DockerExecutor executes Docker CLI commands
type DockerExecutor struct {
	binaryPath string
}

// NewDockerExecutor creates a new Docker command executor
func NewDockerExecutor() *DockerExecutor {
	return &DockerExecutor{
		binaryPath: "docker",
	}
}

// Execute runs a docker command with the given arguments
func (d *DockerExecutor) Execute(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, d.binaryPath, args...)
	return cmd.Output()
}

// RuntimeName returns "docker"
func (d *DockerExecutor) RuntimeName() string {
	return "docker"
}

// ContextInspect executes docker context inspect
func (d *DockerExecutor) ContextInspect(ctx context.Context) ([]byte, error) {
	return d.Execute(ctx, "context", "inspect")
}

// Info executes docker info --format json
func (d *DockerExecutor) Info(ctx context.Context) ([]byte, error) {
	return d.Execute(ctx, "info", "--format", "json")
}

// PodmanExecutor executes Podman CLI commands
type PodmanExecutor struct {
	binaryPath string
}

// NewPodmanExecutor creates a new Podman command executor
func NewPodmanExecutor() *PodmanExecutor {
	return &PodmanExecutor{
		binaryPath: "podman",
	}
}

// Execute runs a podman command with the given arguments
func (p *PodmanExecutor) Execute(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, p.binaryPath, args...)
	return cmd.Output()
}

// RuntimeName returns "podman"
func (p *PodmanExecutor) RuntimeName() string {
	return "podman"
}

// ContextInspect for Podman doesn't have the same context concept as Docker
// Instead, we can return connection information
func (p *PodmanExecutor) ContextInspect(ctx context.Context) ([]byte, error) {
	// Podman doesn't have context like Docker, return connection info instead
	return p.Execute(ctx, "system", "connection", "list", "--format", "json")
}

// Info executes podman info --format json
func (p *PodmanExecutor) Info(ctx context.Context) ([]byte, error) {
	return p.Execute(ctx, "info", "--format", "json")
}

// RuntimeInfo represents system information from container runtime
type RuntimeInfo struct {
	// Common fields
	CgroupVersion    string   `json:"cgroupVersion,omitempty"`
	CgroupController []string `json:"cgroupController,omitempty"`
	Architecture     string   `json:"architecture,omitempty"`
	CPUs             int      `json:"ncpu,omitempty"`
	MemTotal         int64    `json:"memTotal,omitempty"`
	OSType           string   `json:"osType,omitempty"`
	KernelVersion    string   `json:"kernelVersion,omitempty"`
	SecurityOptions  []string `json:"securityOptions,omitempty"`
	
	// Podman-specific structure
	Host *PodmanHostInfo `json:"host,omitempty"`
}

// PodmanHostInfo represents Podman's host information structure
type PodmanHostInfo struct {
	Arch        string              `json:"arch,omitempty"`
	OS          string              `json:"os,omitempty"`
	CPUs        int                 `json:"cpus,omitempty"`
	MemTotal    int64               `json:"memTotal,omitempty"`
	Kernel      string              `json:"kernel,omitempty"`
	CgroupsVersion string           `json:"cgroupsVersion,omitempty"`
	Security    *PodmanSecurityInfo `json:"security,omitempty"`
}

// PodmanSecurityInfo represents Podman's security information
type PodmanSecurityInfo struct {
	ApparmorEnabled     bool   `json:"apparmorEnabled"`
	Rootless           bool   `json:"rootless"`
	SeccompEnabled     bool   `json:"seccompEnabled"`
	SeccompProfilePath string `json:"seccompProfilePath,omitempty"`
	SELinuxEnabled     bool   `json:"selinuxEnabled"`
	Capabilities       string `json:"capabilities,omitempty"`
}

// DockerContextInfo represents Docker context information
type DockerContextInfo struct {
	Name      string `json:"Name"`
	Endpoints struct {
		Docker struct {
			Host string `json:"Host"`
		} `json:"docker"`
	} `json:"Endpoints"`
}

// PodmanConnectionInfo represents Podman connection information
type PodmanConnectionInfo struct {
	Name string `json:"name,omitempty"`
	URI  string `json:"uri,omitempty"`
}

// AutoDetectExecutor detects and returns the appropriate command executor
func AutoDetectExecutor(ctx context.Context) (CommandExecutor, error) {
	// Try Podman first (preferred for rootless)
	podmanExec := NewPodmanExecutor()
	if _, err := podmanExec.Execute(ctx, "version"); err == nil {
		return podmanExec, nil
	}

	// Fall back to Docker
	dockerExec := NewDockerExecutor()
	if _, err := dockerExec.Execute(ctx, "version"); err == nil {
		return dockerExec, nil
	}

	return nil, fmt.Errorf("no container runtime found (tried podman, docker)")
}

// GetRuntimeInfo retrieves and parses runtime information
func GetRuntimeInfo(ctx context.Context, executor CommandExecutor) (*RuntimeInfo, error) {
	infoBytes, err := executor.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime info: %w", err)
	}

	var info RuntimeInfo
	if err := json.Unmarshal(infoBytes, &info); err != nil {
		return nil, fmt.Errorf("failed to parse runtime info: %w", err)
	}

	return &info, nil
}

// GetDockerContexts retrieves Docker context information
func GetDockerContexts(ctx context.Context, executor CommandExecutor) ([]DockerContextInfo, error) {
	if executor.RuntimeName() != "docker" {
		return nil, fmt.Errorf("context inspection only supported for Docker")
	}

	contextBytes, err := executor.ContextInspect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect Docker context: %w", err)
	}

	var contexts []DockerContextInfo
	if err := json.Unmarshal(contextBytes, &contexts); err != nil {
		return nil, fmt.Errorf("failed to parse Docker context: %w", err)
	}

	return contexts, nil
}

// GetPodmanConnections retrieves Podman connection information
func GetPodmanConnections(ctx context.Context, executor CommandExecutor) ([]PodmanConnectionInfo, error) {
	if executor.RuntimeName() != "podman" {
		return nil, fmt.Errorf("connection inspection only supported for Podman")
	}

	connectionBytes, err := executor.ContextInspect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect Podman connections: %w", err)
	}

	var connections []PodmanConnectionInfo
	if err := json.Unmarshal(connectionBytes, &connections); err != nil {
		return nil, fmt.Errorf("failed to parse Podman connections: %w", err)
	}

	return connections, nil
}

// IsRootless determines if the runtime is running in rootless mode
func IsRootless(ctx context.Context, executor CommandExecutor) (bool, error) {
	info, err := GetRuntimeInfo(ctx, executor)
	if err != nil {
		return false, err
	}

	// For Podman, check the host.security.rootless field
	if info.Host != nil && info.Host.Security != nil {
		return info.Host.Security.Rootless, nil
	}

	// For Docker, check security options
	for _, option := range info.SecurityOptions {
		if strings.Contains(strings.ToLower(option), "rootless") {
			return true, nil
		}
	}

	return false, nil
}

// GetSecurityFeatures extracts security features from runtime info
func GetSecurityFeatures(info *RuntimeInfo) []string {
	var features []string
	
	// Handle Podman structure
	if info.Host != nil && info.Host.Security != nil {
		sec := info.Host.Security
		if sec.Rootless {
			features = append(features, "rootless")
		}
		if sec.SeccompEnabled {
			features = append(features, "seccomp")
		}
		if sec.ApparmorEnabled {
			features = append(features, "apparmor")
		}
		if sec.SELinuxEnabled {
			features = append(features, "selinux")
		}
		return features
	}
	
	// Handle Docker structure (security options array)
	for _, option := range info.SecurityOptions {
		optionLower := strings.ToLower(option)
		if strings.Contains(optionLower, "rootless") {
			features = append(features, "rootless")
		}
		if strings.Contains(optionLower, "seccomp") {
			features = append(features, "seccomp")
		}
		if strings.Contains(optionLower, "apparmor") {
			features = append(features, "apparmor")
		}
		if strings.Contains(optionLower, "selinux") {
			features = append(features, "selinux")
		}
	}
	
	return features
}