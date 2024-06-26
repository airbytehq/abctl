package paths

import (
	"github.com/google/go-cmp/cmp"
	"os"
	"path/filepath"
	"testing"
)

func Test_Paths(t *testing.T) {
	t.Run("UserHome", func(t *testing.T) {
		exp, _ := os.UserHomeDir()
		if d := cmp.Diff(exp, UserHome); d != "" {
			t.Errorf("UserHome mismatch (-want +got):\n%s", d)
		}
	})
	t.Run("Airbyte", func(t *testing.T) {
		exp := filepath.Join(UserHome, ".airbyte")
		if d := cmp.Diff(exp, Airbyte); d != "" {
			t.Errorf("Airbyte mismatch (-want +got):\n%s", d)
		}
	})
	t.Run("AbCtl", func(t *testing.T) {
		exp := filepath.Join(UserHome, ".airbyte", "abctl")
		if d := cmp.Diff(exp, AbCtl); d != "" {
			t.Errorf("AbCtl mismatch (-want +got):\n%s", d)
		}
	})
	t.Run("Data", func(t *testing.T) {
		exp := filepath.Join(UserHome, ".airbyte", "abctl", "data")
		if d := cmp.Diff(exp, Data); d != "" {
			t.Errorf("Data mismatch (-want +got):\n%s", d)
		}
	})
}
