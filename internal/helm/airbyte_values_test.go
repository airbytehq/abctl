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
		name    string
		opts    ValuesOpts
		want    string
		wantErr bool
	}{
		{
			name: "default options",
			opts: ValuesOpts{TelemetryUser: "test-user"},
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
			name: "local storage enabled",
			opts: ValuesOpts{TelemetryUser: "test-user", LocalStorage: true},
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
			name: "psql 1.7 enabled",
			opts: ValuesOpts{TelemetryUser: "test-user", EnablePsql17: true},
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
			name: "custom values file merges and overrides",
			opts: ValuesOpts{
				TelemetryUser: "test-user",
				ValuesFile:    filepath.Join(testdataDir, "expected-default.values.yaml"),
			},
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
			name: "invalid values file returns error",
			opts: ValuesOpts{
				TelemetryUser: "test-user",
				ValuesFile:    filepath.Join(testdataDir, "invalid.values.yaml"),
			},
			wantErr: true,
		},
		{
			name: "auth disabled",
			opts: ValuesOpts{TelemetryUser: "test-user", DisableAuth: true},
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
			name: "low resource mode",
			opts: ValuesOpts{TelemetryUser: "test-user", LowResourceMode: true},
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
			name: "insecure cookies disabled",
			opts: ValuesOpts{TelemetryUser: "test-user", InsecureCookies: true},
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
			name: "image pull secret set",
			opts: ValuesOpts{TelemetryUser: "test-user", ImagePullSecret: "mysecret"},
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := BuildAirbyteValues(ctx, tc.opts)
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
