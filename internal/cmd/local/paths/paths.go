package paths

import (
	"os"
	"path/filepath"
)

// UserHome is the user's home directory
var (
	UserHome = func() string {
		h, _ := os.UserHomeDir()
		return h
	}()
	Airbyte = airbyte()
	AbCtl   = abctl()
	Data    = data()
)

func airbyte() string {
	return filepath.Join(UserHome, ".airbyte")
}

func abctl() string {
	return filepath.Join(airbyte(), "abctl")
}

func data() string {
	return filepath.Join(abctl(), "data")
}
