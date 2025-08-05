package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/abctl"
	"github.com/airbytehq/abctl/internal/k8s"
	"github.com/airbytehq/abctl/internal/paths"
	"github.com/airbytehq/abctl/internal/service"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/alecthomas/kong"
	"github.com/google/go-cmp/cmp"
	goHelm "github.com/mittwald/go-helm-client"
)

func TestCheckAirbyteDir(t *testing.T) {
	origDir := paths.Airbyte
	t.Cleanup(func() {
		paths.Airbyte = origDir
	})

	t.Run("no directory", func(t *testing.T) {
		paths.Airbyte = filepath.Join(t.TempDir(), "does-not-exist")
		if err := checkAirbyteDir(); err != nil {
			t.Error("unexpected error", err)
		}
	})

	t.Run("directory with correct permissions", func(t *testing.T) {
		paths.Airbyte = filepath.Join(t.TempDir(), "correct-perms")
		if err := os.MkdirAll(paths.Airbyte, 0o744); err != nil {
			t.Fatal("unable to create test directory", err)
		}
		if err := os.Chmod(paths.Airbyte, 0o744); err != nil {
			t.Fatal("unable to change permissions", err)
		}
		if err := checkAirbyteDir(); err != nil {
			t.Error("unexpected error", err)
		}

		// permissions should be unchanged
		perms, err := os.Stat(paths.Airbyte)
		if err != nil {
			t.Fatal("unable to check permissions", err)
		}
		if d := cmp.Diff(0o744, int(perms.Mode().Perm())); d != "" {
			t.Errorf("permissions mismatch (-want +got):\n%s", d)
		}
	})

	t.Run("directory with higher permissions", func(t *testing.T) {
		paths.Airbyte = filepath.Join(t.TempDir(), "correct-perms")
		if err := os.MkdirAll(paths.Airbyte, 0o777); err != nil {
			t.Fatal("unable to create test directory", err)
		}
		if err := os.Chmod(paths.Airbyte, 0o777); err != nil {
			t.Fatal("unable to change permissions", err)
		}
		if err := checkAirbyteDir(); err != nil {
			t.Error("unexpected error", err)
		}

		// permissions should be unchanged
		perms, err := os.Stat(paths.Airbyte)
		if err != nil {
			t.Fatal("unable to check permissions", err)
		}
		if d := cmp.Diff(0o777, int(perms.Mode().Perm())); d != "" {
			t.Errorf("permissions mismatch (-want +got):\n%s", d)
		}
	})

	t.Run("directory with incorrect permissions", func(t *testing.T) {
		paths.Airbyte = filepath.Join(t.TempDir(), "incorrect-perms")
		if err := os.MkdirAll(paths.Airbyte, 0o200); err != nil {
			t.Fatal("unable to create test directory", err)
		}
		if err := os.Chmod(paths.Airbyte, 0o200); err != nil {
			t.Fatal("unable to change permissions", err)
		}
		// although the permissions are incorrect, checkAirbyteDir should fix them
		if err := checkAirbyteDir(); err != nil {
			t.Fatal("unexpected error", err)
		}

		// permissions should be changed
		perms, err := os.Stat(paths.Airbyte)
		if err != nil {
			t.Fatal("unable to check permissions", err)
		}
		if d := cmp.Diff(0o744, int(perms.Mode().Perm())); d != "" {
			t.Errorf("permissions mismatch (-want +got):\n%s", d)
		}
	})
}

func TestValues_FileDoesntExist(t *testing.T) {
	var root InstallCmd
	k, _ := kong.New(
		&root,
		kong.Name("abctl"),
		kong.Description("Airbyte's command line tool for managing a local Airbyte installation."),
		kong.UsageOnError(),
	)
	_, err := k.Parse([]string{"--values", "/testdata/thisfiledoesnotexist"})
	if err == nil {
		t.Fatal("expected error")
	}
	expect := "--values: stat /testdata/thisfiledoesnotexist: no such file or directory"
	if err.Error() != expect {
		t.Errorf("expected %q but got %q", expect, err)
	}
}

func TestValues_BadYaml(t *testing.T) {
	cmd := InstallCmd{Values: "./testdata/invalid.values.yaml"}
	// Does not need actual clients for tests.
	testFactory := func(kubeConfig, kubeContext string) (k8s.Client, goHelm.Client, error) {
		return nil, nil, nil
	}
	err := cmd.Run(context.Background(), k8s.TestProvider, testFactory, telemetry.NoopClient{})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.HasPrefix(err.Error(), "failed to build values yaml for v2 chart: failed to unmarshal file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInvalidHostFlag_IpAddr(t *testing.T) {
	cmd := InstallCmd{Host: []string{"ok", "1.2.3.4"}}
	// Does not need actual clients for tests.
	testFactory := func(kubeConfig, kubeContext string) (k8s.Client, goHelm.Client, error) {
		return nil, nil, nil
	}
	err := cmd.Run(context.Background(), k8s.TestProvider, testFactory, telemetry.NoopClient{})
	if !errors.Is(err, abctl.ErrIpAddressForHostFlag) {
		t.Errorf("expected ErrIpAddressForHostFlag but got %v", err)
	}
}

func TestInvalidHostFlag_IpAddrWithPort(t *testing.T) {
	cmd := InstallCmd{Host: []string{"ok", "1.2.3.4:8000"}}
	// Does not need actual clients for tests.
	testFactory := func(kubeConfig, kubeContext string) (k8s.Client, goHelm.Client, error) {
		return nil, nil, nil
	}
	err := cmd.Run(context.Background(), k8s.TestProvider, testFactory, telemetry.NoopClient{})
	if !errors.Is(err, abctl.ErrInvalidHostFlag) {
		t.Errorf("expected ErrInvalidHostFlag but got %v", err)
	}
}

func TestInstallOpts(t *testing.T) {
	b, _ := os.ReadFile("./testdata/expected-default.values.yaml")
	cmd := InstallCmd{
		// Don't let the code dynamically resolve the latest chart version.
		Chart: "/test/path/to/chart",
	}
	expect := &service.InstallOpts{
		HelmValuesYaml:  string(b),
		AirbyteChartLoc: "/test/path/to/chart",
		LocalStorage:    true,
		EnablePsql17:    true,
	}
	opts, err := cmd.installOpts(context.Background(), "test-user")
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(expect, opts); d != "" {
		t.Errorf("unexpected error diff (-want +got):\n%s", d)
	}
}
