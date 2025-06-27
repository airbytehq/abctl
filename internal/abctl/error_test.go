package abctl

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLocalError(t *testing.T) {
	f := func() error {
		return &Error{
			help: "help message",
			msg:  "error message",
		}
	}

	err := f()
	var e *Error
	if !errors.As(err, &e) {
		t.Fatal("error should be of type LocalError")
	}

	if d := cmp.Diff("help message", e.Help()); d != "" {
		t.Errorf("help message diff:\n%s", d)
	}
	if d := cmp.Diff("error message", e.Error()); d != "" {
		t.Errorf("error message diff:\n%s", d)
	}
}
