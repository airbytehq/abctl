package helm

import (
	"strings"
	"testing"
)

func TestBuildNginxValues(t *testing.T) {
	values, err := BuildNginxValues(8000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, want := range []string{
		"http: 8000",
		// A connection catalog for a schema with thousands of tables is larger than 10m.
		"proxy-body-size: 50m",
		`proxy-read-timeout: "600"`,
	} {
		if !strings.Contains(values, want) {
			t.Errorf("nginx values missing %q:\n%s", want, values)
		}
	}
}
