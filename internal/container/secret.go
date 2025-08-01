package container

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// DockerConfig represents the structure of a Docker config.json file
type DockerConfig struct {
	Auths map[string]DockerAuth `json:"auths"`
}

// DockerAuth represents authentication information for a Docker registry
type DockerAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

// Secret creates a Docker registry secret in the format expected by Kubernetes
// This function is compatible with both Docker and Podman as it creates standard
// Docker registry authentication format.
func Secret(server, user, pass, email string) ([]byte, error) {
	if server == "" {
		server = "https://index.docker.io/v1/"
	}

	auth := DockerAuth{
		Username: user,
		Password: pass,
		Email:    email,
		Auth:     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, pass))),
	}

	config := DockerConfig{
		Auths: map[string]DockerAuth{
			server: auth,
		},
	}

	return json.Marshal(config)
}