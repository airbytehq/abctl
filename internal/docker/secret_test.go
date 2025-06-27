package docker

import (
	"testing"
)

func Test_Secret(t *testing.T) {
	exp := `{"auths":{"my-registry.example:5000":{"auth":"dGlnZXI6cGFzczEyMzQ=","email":"tiger@acme.example","password":"pass1234","username":"tiger"}}}`
	act, err := Secret("my-registry.example:5000", "tiger", "pass1234", "tiger@acme.example")
	if err != nil {
		t.Fatal(err)
	}
	if exp != string(act) {
		t.Errorf("Secret mismatch:\nwant: %s\ngot:  %s", exp, act)
	}
}
