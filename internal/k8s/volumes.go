package k8s

import (
	"fmt"
	"strings"
)

// errInvalidVolumeMountSpec returns an error for an invalid volume mount spec.
func errInvalidVolumeMountSpec(spec string) error {
	return fmt.Errorf("volume %s is not a valid volume spec, must be <HOST_PATH>:<GUEST_PATH>", spec)
}

// ParseVolumeMounts parses a slice of volume mount specs in the format <HOST_PATH>:<GUEST_PATH>
// and returns a slice of ExtraVolumeMount. Returns an error if any spec is invalid.
func ParseVolumeMounts(specs []string) ([]ExtraVolumeMount, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	mounts := make([]ExtraVolumeMount, len(specs))

	for i, spec := range specs {
		parts := strings.Split(spec, ":")
		if len(parts) != 2 {
			return nil, errInvalidVolumeMountSpec(spec)
		}
		mounts[i] = ExtraVolumeMount{
			HostPath:      parts[0],
			ContainerPath: parts[1],
		}
	}

	return mounts, nil
}
