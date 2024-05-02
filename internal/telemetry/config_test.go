package telemetry

import (
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/oklog/ulid/v2"
	"os"
	"path/filepath"
	"testing"
)

var id = ulid.Make()

func TestLoadConfigFromFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "analytics-")
	if err != nil {
		t.Fatal("could not create temp file", err)
	}
	defer f.Close()

	if _, err := f.WriteString(`# comments
anonymous_user_id: ` + id.String()); err != nil {
		t.Fatal("could not write to temp file", err)
	}

	cnf, err := loadConfigFromFile(f.Name())
	if d := cmp.Diff(nil, err); d != "" {
		t.Error("failed to load file", d)
	}

	if d := cmp.Diff(id.String(), cnf.UserID.String()); d != "" {
		t.Error("id is incorrect", d)
	}
}

func TestLoadConfigFromFile_NoFileReturnsErrNotExist(t *testing.T) {
	_, err := loadConfigFromFile(filepath.Join(t.TempDir(), "dne.yml"))
	if err == nil {
		t.Error("expected an error to be returned")
	}
	// should return a os.ErrNotExist if no file was found
	if d := cmp.Diff(true, errors.Is(err, os.ErrNotExist)); d != "" {
		t.Error("expected os.ErrNotExist", err)
	}
}

func TestLoadConfigFromFile_IncorrectFormatReturnsErr(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "analytics-")
	if err != nil {
		t.Fatal("could not create temp file", err)
	}
	defer f.Close()

	if _, err := f.WriteString(`This is a regular file, not a configuration file!`); err != nil {
		t.Fatal("could not write to temp file", err)
	}

	_, err = loadConfigFromFile(f.Name())
	if err == nil {
		t.Error("expected an error to be returned")
	}
}

func TestLoadConfigFromFile_UnreadableFileReturnsErr(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "analytics-")
	if err != nil {
		t.Fatal("could not create temp file", err)
	}
	defer f.Close()

	// remove read permissions from file
	if err := f.Chmod(0222); err != nil {
		t.Fatal("could not chmod temp file", err)
	}

	_, err = loadConfigFromFile(f.Name())
	if err == nil {
		t.Error("expected an error to be returned")
	}
}

func TestWriteConfigToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deeply", ConfigFile)

	cfg := Config{UserID: ULID(id)}

	if err := writeConfigToFile(path, cfg); err != nil {
		t.Error("failed to create file", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Error("failed to read file", err)
	}

	exp := fmt.Sprintf(`%sanonymous_user_id: %s
`, header, id.String())

	if d := cmp.Diff(exp, string(contents)); d != "" {
		t.Error("contents do not match", d)
	}
}
