package local

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/pterm/pterm"
)

// dockerClient is exposed here primarily for testing purposes.
// A test should override this value to mock out a docker-client.
// If this value is nil, the default docker-client (as returned from defaultDocker) will be utilized.
var dockerClient *docker.Docker

// dockerInstalled checks if docker is installed on the host machine.
// Returns a nil error if docker was successfully detected, otherwise an error will be returned.  Any error returned
// is guaranteed to include the ErrDocker error in the error chain.
func dockerInstalled(ctx context.Context, telClient telemetry.Client) (docker.Version, error) {
	var err error
	if dockerClient == nil {
		if dockerClient, err = docker.New(ctx); err != nil {
			pterm.Error.Println("Unable to create Docker client")
			return docker.Version{}, fmt.Errorf("%w: unable to create client: %w", localerr.ErrDocker, err)
		}
	}

	version, err := dockerClient.Version(ctx)
	if err != nil {
		pterm.Error.Println("Unable to communicate with the Docker daemon")
		return docker.Version{}, fmt.Errorf("%w: %w", localerr.ErrDocker, err)
	}

	telClient.Attr("docker_version", version.Version)
	telClient.Attr("docker_arch", version.Arch)
	telClient.Attr("docker_platform", version.Platform)

	if info, err := dockerClient.Client.Info(ctx); err == nil {
		telClient.Attr("docker_ncpu", fmt.Sprintf("%d", info.NCPU))
		telClient.Attr("docker_memtotal", fmt.Sprintf("%d", info.MemTotal))
	}

	pterm.Success.Println(fmt.Sprintf("Found Docker installation: version %s", version.Version))
	return version, nil

}

// doer interface for testing purposes
type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// httpClient can be overwritten for testing purposes
var httpClient doer = &http.Client{Timeout: 3 * time.Second}

// portAvailable returns a nil error if the port is available, or already is use by Airbyte, otherwise returns an error.
//
// This function works by attempting to establish a tcp listener on a port.
// If we can establish a tcp listener on the port, an additional check is made to see if Airbyte may already be
// bound to that port. If something besides Airbyte is using it, treat this as an inaccessible port.
func portAvailable(ctx context.Context, port int) error {
	if port < 1024 {
		pterm.Warning.Printfln(
			"Availability of port %d cannot be determined, as this is a privileged port (less than 1024).\n"+
				"Installation may not complete successfully",
			port)
		return nil
	}

	// net.Listen doesn't support providing a context
	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("localhost:%d", port))
	if isErrorAddressAlreadyInUse(err) {
		return fmt.Errorf("%w: port %d is already in use", localerr.ErrPort, port)
	}
	if err != nil {
		return fmt.Errorf("%w: unable to determine if port '%d' is available: %w", localerr.ErrPort, port, err)
	}
	// if we're able to bind to the port (and then release it), it should be available
	defer func() {
		_ = listener.Close()
	}()

	return nil
}

func isErrorAddressAlreadyInUse(err error) bool {
	var eOsSyscall *os.SyscallError
	if !errors.As(err, &eOsSyscall) {
		return false
	}
	var errErrno syscall.Errno // doesn't need a "*" (ptr) because it's already a ptr (uintptr)
	if !errors.As(eOsSyscall, &errErrno) {
		return false
	}
	if errors.Is(errErrno, syscall.EADDRINUSE) {
		return true
	}
	const WSAEADDRINUSE = 10048
	if runtime.GOOS == "windows" && errErrno == WSAEADDRINUSE {
		return true
	}
	return false
}

func getPort(ctx context.Context, clusterName string) (int, error) {
	var err error

	if dockerClient == nil {
		dockerClient, err = docker.New(ctx)
		if err != nil {
			return 0, fmt.Errorf("unable to connect to docker: %w", err)
		}
	}

	container := fmt.Sprintf("%s-control-plane", clusterName)

	ci, err := dockerClient.Client.ContainerInspect(ctx, container)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrUnableToInspect, err)
	}
	if ci.State == nil || ci.State.Status != "running" {
		status := "unknown"
		if ci.State != nil {
			status = ci.State.Status
		}
		return 0, ContainerNotRunningError{container, status}
	}

	for _, bindings := range ci.HostConfig.PortBindings {
		for _, ipPort := range bindings {
			if ipPort.HostIP == "0.0.0.0" {
				port, err := strconv.Atoi(ipPort.HostPort)
				if err != nil {
					return 0, InvalidPortError{ipPort.HostPort, err}
				}
				return port, nil
			}
		}
	}

	return 0, fmt.Errorf("%w on container %q", ErrPortNotFound, container)
}

var ErrPortNotFound = errors.New("no matching port found")
var ErrUnableToInspect = errors.New("unable to inspect container")

type ContainerNotRunningError struct {
	Container string
	Status string
}

func (e ContainerNotRunningError) Error() string {
	return fmt.Sprintf("container %q is not running (status = %q)", e.Container, e.Status)
}

type InvalidPortError struct {
	Port string
	Inner error
}
func (e InvalidPortError) Unwrap() error {
	return e.Inner
}
func (e InvalidPortError) Error() string {
	return fmt.Sprintf("unable to convert host port %s to integer: %s", e.Port, e.Inner)
}