package local

import (
	"context"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/docker/docker/api/types"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestDockerInstalled(t *testing.T) {
	t.Cleanup(func() {
		dockerClient = nil
	})

	dockerClient = mockServerVersion{
		serverVersion: func(ctx context.Context) (types.Version, error) {
			return types.Version{
				Platform: struct{ Name string }{Name: "test"},
				Version:  "version",
				Arch:     "arch",
			}, nil
		},
	}

	err := dockerInstalled(context.Background(), telemetry.NoopClient{}, os.TempDir())
	if err != nil {
		t.Error("unexpected error:", err)
	}

}

func TestPortAvailable_Available(t *testing.T) {
	// spin up a listener to find a port and then shut it down to ensure that port is available
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("could not create listener", err)
	}
	p := port(listener.Addr().String())
	if err := listener.Close(); err != nil {
		t.Fatal("could not close listener", err)
	}

	err = portAvailable(context.Background(), p)
	if err != nil {
		t.Error("portAvailable returned unexpected error", err)
	}
}

func TestPortAvailable_Unavailable(t *testing.T) {
	// spin up a listener to find a port, and leave it running so that port is unavailable
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("could not create listener", err)
	}
	defer listener.Close()
	p := port(listener.Addr().String())

	err = portAvailable(context.Background(), p)
	// expecting an error
	if err == nil {
		t.Error("portAvailable returned nil error")
	}
}

// port returns the port from a string value in the format of "ipv4:port" or "ip::v6:port"
func port(s string) int {
	vals := strings.Split(s, ":")
	p, _ := strconv.Atoi(vals[len(vals)-1])
	return p
}

// mocks
type mockServerVersion struct {
	serverVersion func(ctx context.Context) (types.Version, error)
}

func (m mockServerVersion) ServerVersion(ctx context.Context) (types.Version, error) {
	return m.serverVersion(ctx)
}
