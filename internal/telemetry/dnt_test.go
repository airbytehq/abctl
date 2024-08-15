package telemetry

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDNT(t *testing.T) {
	// It's possible the host running this test already has the envVarDNT flag set.
	// Capture its value before removing it so that it can be restored at the end of the test.
	origEnvVars := map[string]string{}
	if origEnvVar, ok := os.LookupEnv(envVarDNT); ok {
		origEnvVars[envVarDNT] = origEnvVar
		if err := os.Unsetenv(envVarDNT); err != nil {
			t.Fatal("unable to unset environment variable:", err)
		}
	}

	test := []struct {
		name     string
		envVar   *string
		expected bool
	}{
		{
			name:     "unset",
			expected: false,
		},
		{
			name:     "empty string",
			envVar:   sPtr(""),
			expected: true,
		},
		{
			name:     "0 value",
			envVar:   sPtr("0"),
			expected: true,
		},
		{
			name:     "any value",
			envVar:   sPtr("any value goes here"),
			expected: true,
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar == nil {
				if err := os.Unsetenv(envVarDNT); err != nil {
					t.Fatal("unable to unset environment variable:", err)
				}
			} else {
				if err := os.Setenv(envVarDNT, *tt.envVar); err != nil {
					t.Fatal("unable to set environment variable:", err)
				}
			}

			if d := cmp.Diff(tt.expected, DNT()); d != "" {
				t.Errorf("DNT() mismatch (-want +got):\n%s", d)
			}
		})
	}

	t.Cleanup(func() {
		for k, v := range origEnvVars {
			_ = os.Setenv(k, v)
		}
	})
}

func sPtr(s string) *string {
	return &s
}
