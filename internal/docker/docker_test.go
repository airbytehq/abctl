package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/airbytehq/abctl/internal/cmd/local/localerr"
	"github.com/airbytehq/abctl/internal/docker/dockertest"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/go-cmp/cmp"
)

// TODO: move this somewhere else.
// This check is done here instead of the dockertest package to
// avoid a circular dependency.
var _ Client = (*dockertest.MockClient)(nil)

func TestNewWithOptions(t *testing.T) {
	tests := []struct {
		name string
		goos string
	}{
		{
			name: "darwin",
			goos: "darwin",
		},
		{
			name: "windows",
			goos: "windows",
		},
		{
			name: "linux",
			goos: "linux",
		},
	}
	expVersion := Version{
		Platform: dockertest.DefaultServerVersion.Platform.Name,
		Arch:     dockertest.DefaultServerVersion.Arch,
		Version:  dockertest.DefaultServerVersion.Version,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pingCalled := 0

			p := mockPinger{
				MockClient: dockertest.NewMockClient(),
				ping: func(ctx context.Context) (types.Ping, error) {
					pingCalled++
					return types.Ping{}, nil
				},
			}

			f := func(opts ...client.Opt) (pinger, error) {
				// as go doesn't have a way to compare to functions, count the number of functions we have
				// and compare those instead
				if d := cmp.Diff(4, len(opts)); d != "" {
					t.Error("unexpected client option count options", d)
				}

				return p, nil
			}

			cli, err := newWithOptions(ctx, f, "darwin")
			if err != nil {
				t.Fatal("failed creating client", err)
			}

			if d := cmp.Diff(1, pingCalled); d != "" {
				t.Error("ping called incorrect number of times", d)
			}

			ver, err := cli.Version(ctx)
			if err != nil {
				t.Fatal("failed fetching version", err)
			}
			if d := cmp.Diff(expVersion, ver); d != "" {
				t.Error("unexpected version", d)
			}
		})
	}
}

func TestNewWithOptions_InitErr(t *testing.T) {
	tests := []struct {
		name        string
		goos        string
		expAttempts int
	}{
		{
			name:        "darwin",
			goos:        "darwin",
			expAttempts: 3,
		},
		{
			name:        "windows",
			goos:        "windows",
			expAttempts: 2,
		},
		{
			name:        "linux",
			goos:        "linux",
			expAttempts: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			attempts := 0

			f := func(opts ...client.Opt) (pinger, error) {
				attempts++
				return nil, errors.New("test error")
			}

			_, err := newWithOptions(ctx, f, tt.goos)
			if err == nil {
				t.Fatal("expected error")
			}
			if d := cmp.Diff(true, errors.Is(err, localerr.ErrDocker)); d != "" {
				t.Error("unexpected error, should be ErrDocker", d)
			}

			// verify the number of attempts to create a new client
			if d := cmp.Diff(tt.expAttempts, attempts); d != "" {
				t.Error("unexpected attempts", d)
			}
		})
	}
}

func TestNewWithOptions_PingErr(t *testing.T) {
	tests := []struct {
		name        string
		goos        string
		expAttempts int
	}{
		{
			name:        "darwin",
			goos:        "darwin",
			expAttempts: 2, // darwin will attempt two different locations
		},
		{
			name:        "windows",
			goos:        "windows",
			expAttempts: 1,
		},
		{
			name:        "linux",
			goos:        "linux",
			expAttempts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			p := mockPinger{
				ping: func(ctx context.Context) (types.Ping, error) {
					return types.Ping{}, errors.New("test error")
				},
			}

			f := func(opts ...client.Opt) (pinger, error) {
				return p, nil
			}

			_, err := newWithOptions(ctx, f, tt.goos)
			if err == nil {
				t.Fatal("expected error")
			}
			if d := cmp.Diff(true, errors.Is(err, localerr.ErrDocker)); d != "" {
				t.Error("unexpected error, should be ErrDocker", d)
			}
		})
	}
}

func TestNewWithOptions_DarwinPingErrFirstAttemptOnly(t *testing.T) {
	ctx := context.Background()
	pingCalled := false

	p := mockPinger{
		ping: func(ctx context.Context) (types.Ping, error) {
			if !pingCalled {
				pingCalled = true
				return types.Ping{}, errors.New("test error")
			}

			return types.Ping{}, nil
		},
	}

	f := func(opts ...client.Opt) (pinger, error) {
		return p, nil
	}

	cli, err := newWithOptions(ctx, f, "darwin")
	if err != nil {
		t.Error("unexpected error", err)
	}
	if cli == nil {
		t.Error("client should not be nil")
	}
}

func TestVersion_Err(t *testing.T) {
	ctx := context.Background()
	p := mockPinger{
		MockClient: dockertest.MockClient{
			FnServerVersion: func(ctx context.Context) (types.Version, error) {
				return types.Version{}, errors.New("test error")
			},
		},
	}

	f := func(opts ...client.Opt) (pinger, error) { return p, nil }

	cli, err := newWithOptions(ctx, f, "darwin")
	if err != nil {
		t.Fatal("failed creating client", err)
	}

	_, err = cli.Version(ctx)
	if err == nil {
		t.Error("expected error")
	}
}

// --- mocks
var _ pinger = (*mockPinger)(nil)

type mockPinger struct {
	dockertest.MockClient
	ping func(ctx context.Context) (types.Ping, error)
}

func (m mockPinger) Ping(ctx context.Context) (types.Ping, error) {
	if m.ping == nil {
		return types.Ping{}, nil
	}

	return m.ping(ctx)
}
