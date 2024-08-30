package k8s

import (
	"errors"
	"fmt"
	"testing"

	"sigs.k8s.io/kind/pkg/exec"
)

func TestFormatKindError(t *testing.T) {
	inner := errors.New("inner")
	runerr := exec.RunError{Command: []string{"one", "two"}, Output: []byte("three"), Inner: inner}
	str := fmt.Errorf("four: %w", formatKindErr(&runerr)).Error()
	expect := `four: command "one two" failed with error: inner: three`
	if str != expect {
		t.Errorf("expected %q but got %q", expect, str)
	}
}
