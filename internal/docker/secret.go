package docker

import (
	containerruntime "github.com/airbytehq/abctl/internal/container"
)

// Secret generates a docker registry secret that can be stored as a k8s secret
// and used to pull images from docker hub (or any other registry) in an
// authenticated manner.
// Deprecated: Use containerruntime.Secret instead
func Secret(server, user, pass, email string) ([]byte, error) {
	return containerruntime.Secret(server, user, pass, email)
}
