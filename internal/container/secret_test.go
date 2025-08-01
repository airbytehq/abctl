package container

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestSecret(t *testing.T) {
	tests := []struct {
		name      string
		server    string
		user      string
		pass      string
		email     string
		wantError bool
	}{
		{
			name:      "basic auth",
			server:    "docker.io",
			user:      "testuser",
			pass:      "testpass",
			email:     "test@example.com",
			wantError: false,
		},
		{
			name:      "empty server defaults to docker hub",
			server:    "",
			user:      "testuser",
			pass:      "testpass",
			email:     "test@example.com",
			wantError: false,
		},
		{
			name:      "custom registry",
			server:    "ghcr.io",
			user:      "testuser",
			pass:      "testpass",
			email:     "test@example.com",
			wantError: false,
		},
		{
			name:      "empty credentials",
			server:    "docker.io",
			user:      "",
			pass:      "",
			email:     "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Secret(tt.server, tt.user, tt.pass, tt.email)
			if (err != nil) != tt.wantError {
				t.Errorf("Secret() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil {
				return // Skip validation if error expected
			}

			// Validate JSON structure
			var config DockerConfig
			if err := json.Unmarshal(got, &config); err != nil {
				t.Errorf("Secret() returned invalid JSON: %v", err)
				return
			}

			// Determine expected server
			expectedServer := tt.server
			if expectedServer == "" {
				expectedServer = "https://index.docker.io/v1/"
			}

			// Check that server exists in auths
			auth, exists := config.Auths[expectedServer]
			if !exists {
				t.Errorf("Secret() missing server %s in auths, got servers: %v", expectedServer, getKeys(config.Auths))
				return
			}

			// Validate auth fields
			if auth.Username != tt.user {
				t.Errorf("Secret() username = %v, want %v", auth.Username, tt.user)
			}
			if auth.Password != tt.pass {
				t.Errorf("Secret() password = %v, want %v", auth.Password, tt.pass)
			}
			if auth.Email != tt.email {
				t.Errorf("Secret() email = %v, want %v", auth.Email, tt.email)
			}

			// Validate base64 auth encoding
			expectedAuth := base64.StdEncoding.EncodeToString([]byte(tt.user + ":" + tt.pass))
			if auth.Auth != expectedAuth {
				t.Errorf("Secret() auth = %v, want %v", auth.Auth, expectedAuth)
			}

			// Validate that auth can be decoded
			decodedAuth, err := base64.StdEncoding.DecodeString(auth.Auth)
			if err != nil {
				t.Errorf("Secret() auth field is not valid base64: %v", err)
				return
			}

			expectedDecoded := tt.user + ":" + tt.pass
			if string(decodedAuth) != expectedDecoded {
				t.Errorf("Secret() decoded auth = %v, want %v", string(decodedAuth), expectedDecoded)
			}
		})
	}
}

func TestSecretFormatCompatibility(t *testing.T) {
	// Test that our format matches what Docker CLI would produce
	server := "docker.io"
	user := "testuser"
	pass := "testpass"
	email := "test@example.com"

	got, err := Secret(server, user, pass, email)
	if err != nil {
		t.Fatalf("Secret() error = %v", err)
	}

	// Parse the result
	var result map[string]interface{}
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatalf("Failed to parse secret JSON: %v", err)
	}

	// Check top-level structure
	auths, ok := result["auths"].(map[string]interface{})
	if !ok {
		t.Fatalf("Secret() missing or invalid 'auths' field")
	}

	serverAuth, ok := auths[server].(map[string]interface{})
	if !ok {
		t.Fatalf("Secret() missing server auth for %s", server)
	}

	// Check required fields exist
	requiredFields := []string{"username", "password", "email", "auth"}
	for _, field := range requiredFields {
		if _, exists := serverAuth[field]; !exists {
			t.Errorf("Secret() missing required field %s", field)
		}
	}

	// Verify no extra fields (for compatibility)
	for field := range serverAuth {
		found := false
		for _, required := range requiredFields {
			if field == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Secret() contains unexpected field %s", field)
		}
	}
}

func TestSecretJSONStructure(t *testing.T) {
	secret, err := Secret("example.com", "user", "pass", "email@example.com")
	if err != nil {
		t.Fatalf("Secret() error = %v", err)
	}

	// Verify it's valid JSON
	if !json.Valid(secret) {
		t.Errorf("Secret() did not return valid JSON")
	}

	// Verify it's properly formatted (no extra whitespace, etc.)
	var compact interface{}
	if err := json.Unmarshal(secret, &compact); err != nil {
		t.Fatalf("Failed to unmarshal secret: %v", err)
	}

	compacted, err := json.Marshal(compact)
	if err != nil {
		t.Fatalf("Failed to marshal secret: %v", err)
	}

	if string(secret) != string(compacted) {
		t.Errorf("Secret() is not compact JSON")
	}
}

// getKeys returns the keys from a map for testing purposes
func getKeys(m map[string]DockerAuth) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestSecretEmptyFields(t *testing.T) {
	// Test with various empty field combinations
	tests := []struct {
		name   string
		server string
		user   string
		pass   string
		email  string
	}{
		{"all empty", "", "", "", ""},
		{"only user", "", "user", "", ""},
		{"only pass", "", "", "pass", ""},
		{"only email", "", "", "", "email@example.com"},
		{"user and pass", "", "user", "pass", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := Secret(tt.server, tt.user, tt.pass, tt.email)
			if err != nil {
				t.Errorf("Secret() with empty fields failed: %v", err)
				return
			}

			// Should still produce valid JSON
			var config DockerConfig
			if err := json.Unmarshal(secret, &config); err != nil {
				t.Errorf("Secret() with empty fields produced invalid JSON: %v", err)
			}

			// Should have default server if empty
			expectedServer := tt.server
			if expectedServer == "" {
				expectedServer = "https://index.docker.io/v1/"
			}

			if _, exists := config.Auths[expectedServer]; !exists {
				t.Errorf("Secret() missing expected server %s", expectedServer)
			}
		})
	}
}