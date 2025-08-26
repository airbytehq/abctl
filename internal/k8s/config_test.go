package k8s

import (
	"testing"

	"github.com/airbytehq/abctl/internal/abctl"
	"github.com/google/go-cmp/cmp"
)

func TestAbctlConfigFromData(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]string
		want    *abctl.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful extraction",
			data: map[string]string{
				"AIRBYTE_API_HOST": "http://localhost:8001/api/public",
				"AIRBYTE_URL":      "https://local.airbyte.dev",
				"AB_AIRBYTE_AUTH_IDENTITY_PROVIDER_OIDC_ENDPOINTS_AUTHORIZATION_SERVER_ENDPOINT": "https://keycloak.internal.airbyte.dev/auth/realms/airbyte",
			},
			want: &abctl.Config{
				AirbyteAPIHost: "http://localhost:8001/api/public",
				AirbyteURL:     "https://local.airbyte.dev",
				AirbyteAuthURL: "https://keycloak.internal.airbyte.dev/auth/realms/airbyte",
			},
			wantErr: false,
		},
		{
			name: "missing AIRBYTE_API_HOST",
			data: map[string]string{
				"AIRBYTE_URL":    "https://local.airbyte.dev",
				"SOME_OTHER_VAR": "value",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "required field AIRBYTE_API_HOST not found or empty",
		},
		{
			name: "missing AIRBYTE_URL",
			data: map[string]string{
				"AIRBYTE_API_HOST": "http://localhost:8001/api/public",
				"SOME_OTHER_VAR":   "value",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "required field AIRBYTE_URL not found or empty",
		},
		{
			name: "empty AIRBYTE_API_HOST",
			data: map[string]string{
				"AIRBYTE_API_HOST": "",
				"AIRBYTE_URL":      "https://local.airbyte.dev",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "required field AIRBYTE_API_HOST not found or empty",
		},
		{
			name: "empty AIRBYTE_URL",
			data: map[string]string{
				"AIRBYTE_API_HOST": "http://localhost:8001/api/public",
				"AIRBYTE_URL":      "",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "required field AIRBYTE_URL not found or empty",
		},
		{
			name: "successful extraction without auth URL",
			data: map[string]string{
				"AIRBYTE_API_HOST": "http://localhost:8001/api/public",
				"AIRBYTE_URL":      "https://local.airbyte.dev",
			},
			want: &abctl.Config{
				AirbyteAPIHost: "http://localhost:8001/api/public",
				AirbyteURL:     "https://local.airbyte.dev",
				AirbyteAuthURL: "", // Should be empty when not provided
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AbctlConfigFromData(tt.data)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("AbctlConfigFromData() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("AbctlConfigFromData() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			
			if err != nil {
				t.Errorf("AbctlConfigFromData() unexpected error = %v", err)
				return
			}
			
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("AbctlConfigFromData() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}