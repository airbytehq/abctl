package docker

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_Secret(t *testing.T) {
	exp := `{"auths":{"my-registry.example:5000":{"username":"tiger","password":"pass1234","email":"tiger@acme.example","auth":"dGlnZXI6cGFzczEyMzQ="}}}`
	act, err := Secret("my-registry.example:5000", "tiger", "pass1234", "tiger@acme.example")
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(exp, string(act)); d != "" {
		t.Errorf("Secret mismatch (-want +got):\n%s", d)
	}
}
