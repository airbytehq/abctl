package images

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestManifestCmd(t *testing.T) {
	cmd := ManifestCmd{
		ChartVersion: "1.1.0",
	}
	actual, err := cmd.findAirbyteImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expect := []string{
		"airbyte/bootloader:1.1.0",
		"airbyte/connector-builder-server:1.1.0",
		"airbyte/cron:1.1.0",
		"airbyte/db:1.1.0",
		"airbyte/mc",
		"airbyte/server:1.1.0",
		"airbyte/webapp:1.1.0",
		"airbyte/worker:1.1.0",
		"airbyte/workload-api-server:1.1.0",
		"airbyte/workload-launcher:1.1.0",
		"bitnami/kubectl:1.28.9",
		"busybox",
		"minio/minio:RELEASE.2023-11-20T22-40-07Z",
		"temporalio/auto-setup:1.23.0",
	}
	compareList(t, expect, actual)
}

func TestManifestCmd_Enterprise(t *testing.T) {
	cmd := ManifestCmd{
		ChartVersion: "1.1.0",
		Values:       "testdata/enterprise.values.yaml",
	}
	actual, err := cmd.findAirbyteImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expect := []string{
		"airbyte/bootloader:1.1.0",
		"airbyte/connector-builder-server:1.1.0",
		"airbyte/cron:1.1.0",
		"airbyte/db:1.1.0",
		"airbyte/keycloak-setup:1.1.0",
		"airbyte/keycloak:1.1.0",
		"airbyte/mc",
		"airbyte/server:1.1.0",
		"airbyte/webapp:1.1.0",
		"airbyte/worker:1.1.0",
		"airbyte/workload-api-server:1.1.0",
		"airbyte/workload-launcher:1.1.0",
		"bitnami/kubectl:1.28.9",
		"busybox",
		"curlimages/curl:8.1.1",
		"minio/minio:RELEASE.2023-11-20T22-40-07Z",
		"postgres:13-alpine",
		"temporalio/auto-setup:1.23.0",
	}
	compareList(t, expect, actual)
}

func TestManifestCmd_Nightly(t *testing.T) {
	cmd := ManifestCmd{
		// This version includes chart fixes that expose images more consistently and completely.
		ChartVersion: "1.1.0-nightly-1728428783-9025e1a46e",
		Values:       "testdata/enterprise.values.yaml",
	}
	actual, err := cmd.findAirbyteImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expect := []string{
		"airbyte/bootloader:nightly-1728428783-9025e1a46e",
		"airbyte/connector-builder-server:nightly-1728428783-9025e1a46e",
		"airbyte/connector-sidecar:nightly-1728428783-9025e1a46e",
		"airbyte/container-orchestrator:nightly-1728428783-9025e1a46e",
		"airbyte/cron:nightly-1728428783-9025e1a46e",
		"airbyte/db:nightly-1728428783-9025e1a46e",
		"airbyte/keycloak-setup:nightly-1728428783-9025e1a46e",
		"airbyte/keycloak:nightly-1728428783-9025e1a46e",
		"airbyte/mc:latest",
		"airbyte/server:nightly-1728428783-9025e1a46e",
		"airbyte/webapp:nightly-1728428783-9025e1a46e",
		"airbyte/worker:nightly-1728428783-9025e1a46e",
		"airbyte/workload-api-server:nightly-1728428783-9025e1a46e",
		"airbyte/workload-init-container:nightly-1728428783-9025e1a46e",
		"airbyte/workload-launcher:nightly-1728428783-9025e1a46e",
		"bitnami/kubectl:1.28.9",
		"busybox:1.35",
		"busybox:latest",
		"curlimages/curl:8.1.1",
		"minio/minio:RELEASE.2023-11-20T22-40-07Z",
		"postgres:13-alpine",
		"temporalio/auto-setup:1.23.0",
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
