package airbyte

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/airbytehq/abctl/internal/abctl"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestBuildEnterpriseConfig(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		instanceConfig *InstanceConfiguration
		want           *abctl.Config
		wantErr        bool
	}{
		{
			name:     "successful with auth URL",
			endpoint: "https://enterprise.example.com",
			instanceConfig: &InstanceConfiguration{
				Edition: "enterprise",
				Auth: struct {
					Mode                   string `json:"mode"`
					AuthorizationServerUrl string `json:"authorizationServerUrl,omitempty"`
				}{
					Mode:                   "oidc",
					AuthorizationServerUrl: "https://auth.example.com/realms/airbyte",
				},
			},
			want: &abctl.Config{
				AirbyteAPIHost: "https://enterprise.example.com/api/v1",
				AirbyteURL:     "https://enterprise.example.com",
				AirbyteAuthURL: "https://auth.example.com/realms/airbyte",
				OIDCClientID:   "airbyte-webapp",
				Edition:        "enterprise",
			},
		},
		{
			name:     "successful with default auth URL",
			endpoint: "https://enterprise.example.com/",
			instanceConfig: &InstanceConfiguration{
				Edition: "community",
				Auth: struct {
					Mode                   string `json:"mode"`
					AuthorizationServerUrl string `json:"authorizationServerUrl,omitempty"`
				}{
					Mode:                   "oidc",
					AuthorizationServerUrl: "",
				},
			},
			want: &abctl.Config{
				AirbyteAPIHost: "https://enterprise.example.com/api/v1",
				AirbyteURL:     "https://enterprise.example.com",
				AirbyteAuthURL: "https://enterprise.example.com/auth/realms/airbyte",
				OIDCClientID:   "airbyte-webapp",
				Edition:        "community",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildEnterpriseConfig(tt.endpoint, tt.instanceConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildEnterpriseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("BuildEnterpriseConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateEnterpriseEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		serverResponse interface{}
		serverStatus   int
		want           *InstanceConfiguration
		wantErr        bool
	}{
		{
			name:     "successful OIDC validation",
			endpoint: "https://enterprise.example.com",
			serverResponse: InstanceConfiguration{
				Edition: "enterprise",
				Auth: struct {
					Mode                   string `json:"mode"`
					AuthorizationServerUrl string `json:"authorizationServerUrl,omitempty"`
				}{
					Mode:                   "oidc",
					AuthorizationServerUrl: "https://auth.example.com/realms/airbyte",
				},
			},
			serverStatus: http.StatusOK,
			want: &InstanceConfiguration{
				Edition: "enterprise",
				Auth: struct {
					Mode                   string `json:"mode"`
					AuthorizationServerUrl string `json:"authorizationServerUrl,omitempty"`
				}{
					Mode:                   "oidc",
					AuthorizationServerUrl: "https://auth.example.com/realms/airbyte",
				},
			},
		},
		{
			name:     "non-OIDC auth mode",
			endpoint: "https://enterprise.example.com",
			serverResponse: InstanceConfiguration{
				Edition: "community",
				Auth: struct {
					Mode                   string `json:"mode"`
					AuthorizationServerUrl string `json:"authorizationServerUrl,omitempty"`
				}{
					Mode: "none",
				},
			},
			serverStatus: http.StatusOK,
			want:         nil,
			wantErr:      true,
		},
		{
			name:           "server error",
			endpoint:       "https://enterprise.example.com",
			serverResponse: map[string]string{"error": "Internal Server Error"},
			serverStatus:   http.StatusInternalServerError,
			want:           nil,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/instance_configuration" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(tt.serverStatus)
				err := json.NewEncoder(w).Encode(tt.serverResponse)
				require.NoError(t, err, "encoding json response")
			}))
			defer server.Close()

			// Use test server URL
			testEndpoint := server.URL
			if tt.endpoint != "" {
				testEndpoint = server.URL
			}

			got, err := ValidateEnterpriseEndpoint(context.Background(), testEndpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEnterpriseEndpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ValidateEnterpriseEndpoint() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateEndpointURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid https URL",
			input:   "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			input:   "http://localhost:8000",
			wantErr: false,
		},
		{
			name:    "empty URL",
			input:   "",
			wantErr: true,
		},
		{
			name:    "missing scheme",
			input:   "example.com",
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			input:   "ftp://example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpointURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEndpointURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCompanyIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid identifier",
			input:   "my-company",
			wantErr: false,
		},
		{
			name:    "valid identifier with numbers",
			input:   "company-123",
			wantErr: false,
		},
		{
			name:    "empty identifier",
			input:   "",
			wantErr: true,
		},
		{
			name:    "too short",
			input:   "a",
			wantErr: true,
		},
		{
			name:    "contains spaces",
			input:   "my company",
			wantErr: true,
		},
		{
			name:    "contains special characters",
			input:   "my@company",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCompanyIdentifier(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCompanyIdentifier() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
