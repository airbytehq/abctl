package helm

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBuildAirbyteValues(t *testing.T) {
	testdataDir := filepath.Join("..", "cmd", "local", "testdata")
	cases := []struct {
		name         string
		opts         ValuesOpts
		chartVersion string
		want         string
		wantErr      bool
	}{
		{
			name:         "v1: default options",
			opts:         ValuesOpts{TelemetryUser: "test-user"},
			chartVersion: "1.9.9",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
global:
    auth:
        enabled: true
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
`,
		},
		{
			name:         "v1: local storage enabled",
			opts:         ValuesOpts{TelemetryUser: "test-user", LocalStorage: true},
			chartVersion: "1.9.9",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
global:
    auth:
        enabled: true
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
    storage:
        type: local
`,
		},
		{
			name:         "v1: psql 1.7 enabled",
			opts:         ValuesOpts{TelemetryUser: "test-user", EnablePsql17: true},
			chartVersion: "1.9.9",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
global:
    auth:
        enabled: true
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
postgresql:
    image:
        tag: 1.7.0-17
`,
		},
		{
			name: "v1: custom values file merges and overrides",
			opts: ValuesOpts{
				TelemetryUser: "test-user",
				ValuesFile:    filepath.Join(testdataDir, "expected-default.values.yaml"),
			},
			chartVersion: "1.9.9",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
global:
    auth:
        enabled: true
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
    storage:
        type: local
postgresql:
    image:
        tag: 1.7.0-17
`,
		},
		{
			name: "v1: invalid values file returns error",
			opts: ValuesOpts{
				TelemetryUser: "test-user",
				ValuesFile:    filepath.Join(testdataDir, "invalid.values.yaml"),
			},
			chartVersion: "1.9.9",
			wantErr:      true,
		},
		{
			name:         "v1: auth disabled",
			opts:         ValuesOpts{TelemetryUser: "test-user", DisableAuth: true},
			chartVersion: "1.9.9",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
global:
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
`,
		},
		{
			name:         "v1: low resource mode",
			opts:         ValuesOpts{TelemetryUser: "test-user", LowResourceMode: true},
			chartVersion: "1.9.9",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
connector-builder-server:
    enabled: false
global:
    auth:
        enabled: true
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
            requests:
                cpu: "0"
                memory: "0"
server:
    env_vars:
        JOB_RESOURCE_VARIANT_OVERRIDE: lowresource
workload-launcher:
    env_vars:
        CHECK_JOB_MAIN_CONTAINER_CPU_REQUEST: "0"
        CHECK_JOB_MAIN_CONTAINER_MEMORY_REQUEST: "0"
        DISCOVER_JOB_MAIN_CONTAINER_CPU_REQUEST: "0"
        DISCOVER_JOB_MAIN_CONTAINER_MEMORY_REQUEST: "0"
        SIDECAR_MAIN_CONTAINER_CPU_REQUEST: "0"
        SIDECAR_MAIN_CONTAINER_MEMORY_REQUEST: "0"
        SPEC_JOB_MAIN_CONTAINER_CPU_REQUEST: "0"
        SPEC_JOB_MAIN_CONTAINER_MEMORY_REQUEST: "0"
`,
		},
		{
			name:         "v1: insecure cookies disabled",
			opts:         ValuesOpts{TelemetryUser: "test-user", InsecureCookies: true},
			chartVersion: "1.9.9",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
global:
    auth:
        cookieSecureSetting: '"false"'
        enabled: true
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
`,
		},
		{
			name:         "v1: image pull secret set",
			opts:         ValuesOpts{TelemetryUser: "test-user", ImagePullSecret: "mysecret"},
			chartVersion: "1.9.9",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
global:
    auth:
        enabled: true
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    imagePullSecrets[0]:
        name: mysecret
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
`,
		},
		{
			name:         "v2: default options",
			opts:         ValuesOpts{TelemetryUser: "test-user"},
			chartVersion: "2.0.0",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
global:
    auth:
        enabled: true
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
server:
    env_vars:
        WEBAPP_URL: http://airbyte-abctl-airbyte-server-svc:80
`,
		},
		{
			name:         "v2: all options enabled",
			opts:         ValuesOpts{TelemetryUser: "test-user", LocalStorage: true, EnablePsql17: true, LowResourceMode: true, InsecureCookies: true, ImagePullSecret: "mysecret", DisableAuth: false},
			chartVersion: "2.0.0",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
connectorBuilderServer:
    enabled: false
global:
    auth:
        cookieSecureSetting: '"false"'
        enabled: true
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    imagePullSecrets[0]:
        name: mysecret
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
            requests:
                cpu: "0"
                memory: "0"
    storage:
        type: local
postgresql:
    image:
        tag: 1.7.0-17
server:
    env_vars:
        JOB_RESOURCE_VARIANT_OVERRIDE: lowresource
        WEBAPP_URL: http://airbyte-abctl-airbyte-server-svc:80
workloadLauncher:
    env_vars:
        CHECK_JOB_MAIN_CONTAINER_CPU_REQUEST: "0"
        CHECK_JOB_MAIN_CONTAINER_MEMORY_REQUEST: "0"
        DISCOVER_JOB_MAIN_CONTAINER_CPU_REQUEST: "0"
        DISCOVER_JOB_MAIN_CONTAINER_MEMORY_REQUEST: "0"
        SIDECAR_MAIN_CONTAINER_CPU_REQUEST: "0"
        SIDECAR_MAIN_CONTAINER_MEMORY_REQUEST: "0"
        SPEC_JOB_MAIN_CONTAINER_CPU_REQUEST: "0"
        SPEC_JOB_MAIN_CONTAINER_MEMORY_REQUEST: "0"
`,
		},
		{
			name:         "empty chart version",
			opts:         ValuesOpts{TelemetryUser: "test-user"},
			chartVersion: "",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
global:
    auth:
        enabled: true
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
`,
		},
		{
			name:         "v2 pre-release version",
			opts:         ValuesOpts{TelemetryUser: "test-user"},
			chartVersion: "2.0.0-alpha.1",
			want: `airbyte-bootloader:
    env_vars:
        PLATFORM_LOG_FORMAT: json
global:
    auth:
        enabled: true
    env_vars:
        AIRBYTE_INSTALLATION_ID: test-user
    jobs:
        resources:
            limits:
                cpu: "3"
                memory: 4Gi
`,
		},
		{
			name: "v2: invalid values file returns error",
			opts: ValuesOpts{
				TelemetryUser: "test-user",
				ValuesFile:    filepath.Join(testdataDir, "invalid.values.yaml"),
			},
			chartVersion: "2.0.0",
			wantErr:      true,
		},
		{
			name: "v2: values file not found returns error",
			opts: ValuesOpts{
				TelemetryUser: "test-user",
				ValuesFile:    filepath.Join(testdataDir, "nonexistent.values.yaml"),
			},
			chartVersion: "2.0.0",
			wantErr:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := BuildAirbyteValues(ctx, tc.opts, tc.chartVersion)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			var wantMap, gotMap map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(tc.want), &wantMap), "unmarshal want yaml")
			require.NoError(t, yaml.Unmarshal([]byte(got), &gotMap), "unmarshal got yaml")
			require.Equal(t, wantMap, gotMap)
		})
	}
}
