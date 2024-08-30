package localerr

import "errors"

var (
	// ErrAirbyteDir is returned anytime an there is an issue in accessing the paths.Airbyte directory.
	ErrAirbyteDir = errors.New("airbyte directory is inaccessible")

	// ErrClusterNotFound is returned in the event that no cluster was located.
	ErrClusterNotFound = errors.New("no existing cluster found")

	// ErrDocker is returned anytime an error occurs when attempting to communicate with docker.
	ErrDocker = errors.New("error communicating with docker")

	// ErrKubernetes is returned anytime an error occurs when attempting to communicate with the kubernetes cluster.
	ErrKubernetes = errors.New("error communicating with kubernetes")

	// ErrIngress is returned in the event that ingress configuration failed.
	ErrIngress = errors.New("error configuring ingress")

	// ErrPort is returned in the event that the requested port is unavailable.
	ErrPort = errors.New("error verifying port availability")
)
