package container

import (
	"context"
	"fmt"
	"strings"
)

// Capabilities represents the capabilities of a container runtime
type Capabilities struct {
	SupportsRootless   bool
	SupportsCgroups    bool
	SupportsSeccomp    bool
	SupportsAppArmor   bool
	SupportsSELinux    bool
	CgroupVersion      string
	SecurityOptions    []string
}

// GetCapabilities retrieves the capabilities of the container runtime
func (cr *ContainerRuntime) GetCapabilities(ctx context.Context) (*Capabilities, error) {
	if cr.provider == nil {
		return nil, fmt.Errorf("no provider available")
	}

	info, err := cr.provider.GetSystemInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	capabilities := &Capabilities{
		CgroupVersion:   info.CgroupVersion,
		SecurityOptions: info.SecurityOptions,
	}

	// Analyze security options
	for _, option := range info.SecurityOptions {
		optionLower := strings.ToLower(option)
		switch {
		case strings.Contains(optionLower, "rootless"):
			capabilities.SupportsRootless = true
		case strings.Contains(optionLower, "seccomp"):
			capabilities.SupportsSeccomp = true
		case strings.Contains(optionLower, "apparmor"):
			capabilities.SupportsAppArmor = true
		case strings.Contains(optionLower, "selinux"):
			capabilities.SupportsSELinux = true
		}
	}

	// Check cgroup support
	capabilities.SupportsCgroups = info.CgroupVersion != ""

	return capabilities, nil
}

// SupportsFeature checks if the runtime supports a specific feature
func (c *Capabilities) SupportsFeature(feature string) bool {
	switch strings.ToLower(feature) {
	case "rootless":
		return c.SupportsRootless
	case "cgroups":
		return c.SupportsCgroups
	case "seccomp":
		return c.SupportsSeccomp
	case "apparmor":
		return c.SupportsAppArmor
	case "selinux":
		return c.SupportsSELinux
	default:
		return false
	}
}

// GetRuntimeFeatures returns a list of supported features as strings
func (c *Capabilities) GetRuntimeFeatures() []string {
	var features []string
	
	if c.SupportsRootless {
		features = append(features, "rootless")
	}
	if c.SupportsCgroups {
		features = append(features, "cgroups")
	}
	if c.SupportsSeccomp {
		features = append(features, "seccomp")
	}
	if c.SupportsAppArmor {
		features = append(features, "apparmor")
	}
	if c.SupportsSELinux {
		features = append(features, "selinux")
	}
	
	return features
}

// IsCompatibleWith checks if this runtime is compatible with another runtime's requirements
func (c *Capabilities) IsCompatibleWith(required *Capabilities) bool {
	// Check if all required features are supported
	if required.SupportsRootless && !c.SupportsRootless {
		return false
	}
	if required.SupportsCgroups && !c.SupportsCgroups {
		return false
	}
	if required.SupportsSeccomp && !c.SupportsSeccomp {
		return false
	}
	if required.SupportsAppArmor && !c.SupportsAppArmor {
		return false
	}
	if required.SupportsSELinux && !c.SupportsSELinux {
		return false
	}
	
	return true
}