package images

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestManifestCmd(t *testing.T) {
	cmd := ManifestCmd{
		ChartVersion: "2.0.5",
	}
	actual, err := cmd.findAirbyteImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expect := []string{
		"airbyte/async-profiler:1.7.0",
		"airbyte/bootloader:1.7.0",
		"airbyte/connector-builder-server:1.7.0",
		"airbyte/connector-sidecar:1.7.0",
		"airbyte/container-orchestrator:1.7.0",
		"airbyte/cron:1.7.0",
		"airbyte/db:1.7.0",
		"airbyte/server:1.7.0",
		"airbyte/utils:1.7.0",
		"airbyte/webapp:1.7.0",
		"airbyte/worker:1.7.0",
		"airbyte/workload-api-server:1.7.0",
		"airbyte/workload-init-container:1.7.0",
		"airbyte/workload-launcher:1.7.0",
		"minio/minio:RELEASE.2023-11-20T22-40-07Z",
		"temporalio/auto-setup:1.27.2",
	}
	compareList(t, expect, actual)
}

func TestManifestCmd_Enterprise(t *testing.T) {
	cmd := ManifestCmd{
		ChartVersion: "2.0.5",
		Values:       "testdata/enterprise.values.yaml",
	}
	actual, err := cmd.findAirbyteImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expect := []string{
		"airbyte/async-profiler:1.7.0",
		"airbyte/bootloader:1.7.0",
		"airbyte/connector-builder-server:1.7.0",
		"airbyte/connector-sidecar:1.7.0",
		"airbyte/container-orchestrator:1.7.0",
		"airbyte/cron:1.7.0",
		"airbyte/db:1.7.0",
		"airbyte/server:1.7.0",
		"airbyte/utils:1.7.0",
		"airbyte/webapp:1.7.0",
		"airbyte/worker:1.7.0",
		"airbyte/workload-api-server:1.7.0",
		"airbyte/workload-init-container:1.7.0",
		"airbyte/workload-launcher:1.7.0",
		"minio/minio:RELEASE.2023-11-20T22-40-07Z",
		"temporalio/auto-setup:1.27.2",
	}
	compareList(t, expect, actual)
}

func compareList(t *testing.T, expect, actual []string) {
	t.Helper()
	sort.Strings(expect)
	sort.Strings(actual)
	if d := cmp.Diff(expect, actual); d != "" {
		t.Error(d)
	}
}
