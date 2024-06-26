package paths

import (
	"os"
	"path/filepath"
)

var (
	// UserHome is the user's home directory
	UserHome = func() string {
		h, _ := os.UserHomeDir()
		return h
	}()
	// Airbyte is the full path to the ~/.airbyte directory
	Airbyte = airbyte()
	// AbCtl is the full path to the ~/.airbyte/abctl directory
	AbCtl = abctl()
	// Data is the full path to the ~/.airbyte/abctl/data directory
	Data = data()
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
