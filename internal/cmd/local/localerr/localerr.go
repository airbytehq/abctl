package localerr

var _ error = (*LocalError)(nil)

// LocalError adds a user-friendly help message to specific errors.
type LocalError struct {
	help string
	msg  string
	code string
}

// Help will displayed to the user if this specific error is ever returned.
func (e *LocalError) Help() string {
	return e.help
}

// Error returns the error message.
func (e *LocalError) Error() string {
	return e.msg
}

func (e *LocalError) ErrorCode() string {
	return e.code
}

var (
	// ErrAirbyteDir is returned anytime an there is an issue in accessing the paths.Airbyte directory.
	ErrAirbyteDir = &LocalError{
		code: "ErrAirbyteDir",
		msg: "airbyte directory is inaccessible",
		help: `The ~/.airbyte directory is inaccessible.
You may need to remove this directory before trying your command again.`,
	}

	// ErrClusterNotFound is returned in the event that no cluster was located.
	ErrClusterNotFound = &LocalError{
		code: "ErrClusterNotFound",
		msg: "no existing cluster found",
		help: `No cluster was found. If this is unexpected,
you may need to run the "local install" command again.`,
	}

	// ErrDocker is returned anytime an error occurs when attempting to communicate with docker.
	ErrDocker = &LocalError{
		code: "ErrDocker",
		msg: "error communicating with docker",
		help: `An error occurred while communicating with the Docker daemon.
Ensure that Docker is running and is accessible.  You may need to upgrade to a newer version of Docker.
For additional help please visit https://docs.docker.com/get-docker/`,
	}

	// ErrKubernetes is returned anytime an error occurs when attempting to communicate with the kubernetes cluster.
	ErrKubernetes = &LocalError{
		code: "ErrKubernetes",
		msg: "error communicating with kubernetes",
		help: `An error occurred while communicating with the Kubernetes cluster.
		If this error persists, you may need to run the uninstall command before attempting to run
		the install command again.`,
	}

	// ErrIngress is returned in the event that ingress configuration failed.
	ErrIngress = &LocalError{
		code: "ErrIngress",
		msg: "error configuring ingress",
		help: `An error occurred while configuring ingress.
This could be in indication that the ingress port is already in use by a different application.
The ingress port can be changed by passing the flag --port.`,
	}

	// ErrPort is returned in the event that the requested port is unavailable.
	ErrPort = &LocalError{
		code: "ErrPort",
		msg: "error verifying port availability",
		help: `An error occurred while verifying if the request port is available.
This could be in indication that the ingress port is already in use by a different application.
The ingress port can be changed by passing the flag --port.`,
	}
)
