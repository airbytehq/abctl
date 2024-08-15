package paths

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_Paths(t *testing.T) {
	t.Run("FileKubeconfig", func(t *testing.T) {
		if d := cmp.Diff("abctl.kubeconfig", FileKubeconfig); d != "" {
			t.Errorf("FileKubeconfig mismatch (-want +got):\n%s", d)
		}
	})

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

	t.Run("Kubeconfig", func(t *testing.T) {
		exp := filepath.Join(UserHome, ".airbyte", "abctl", "abctl.kubeconfig")
		if d := cmp.Diff(exp, Kubeconfig); d != "" {
			t.Errorf("Kubeconfig mismatch (-want +got):\n%s", d)
		}
	})
}
