package telemetry

import (
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var ulidID = ulid.Make()
var uuidID = uuid.New()

func TestLoadConfigWithULID(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "analytics-")
		if err != nil {
			t.Fatal("could not create temp file", err)
		}
		defer f.Close()

		if _, err := f.WriteString(`# comments
anonymous_user_id: ` + ulidID.String()); err != nil {
			t.Fatal("could not write to temp file", err)
		}

		cnf, err := loadConfigFromFile(f.Name())
		if d := cmp.Diff(nil, err); d != "" {
			t.Error("failed to load file", d)
		}

		if d := cmp.Diff(ulidID.String(), cnf.UserID.String()); d != "" {
			t.Error("id is incorrect", d)
		}
	})

	t.Run("no file returns err", func(t *testing.T) {
		_, err := loadConfigFromFile(filepath.Join(t.TempDir(), "dne.yml"))
		if err == nil {
			t.Error("expected an error to be returned")
		}
		// should return a os.ErrNotExist if no file was found
		if d := cmp.Diff(true, errors.Is(err, os.ErrNotExist)); d != "" {
			t.Error("expected os.ErrNotExist", err)
		}
	})

	t.Run("incorrect format returns err", func(t *testing.T) {
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
	})

	t.Run("unreadable file returns err", func(t *testing.T) {
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
	})
}

func TestLoadConfigWithUUID(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "analytics-")
		if err != nil {
			t.Fatal("could not create temp file", err)
		}
		defer f.Close()

		if _, err := f.WriteString(`# comments
anonymous_user_uuid: ` + uuidID.String()); err != nil {
			t.Fatal("could not write to temp file", err)
		}

		cnf, err := loadConfigFromFile(f.Name())
		if d := cmp.Diff(nil, err); d != "" {
			t.Error("failed to load file", d)
		}

		if d := cmp.Diff(uuidID.String(), cnf.UserUUID.String()); d != "" {
			t.Error("id is incorrect", d)
		}
	})

	t.Run("no file returns err", func(t *testing.T) {
		_, err := loadConfigFromFile(filepath.Join(t.TempDir(), "dne.yml"))
		if err == nil {
			t.Error("expected an error to be returned")
		}
		// should return a os.ErrNotExist if no file was found
		if d := cmp.Diff(true, errors.Is(err, os.ErrNotExist)); d != "" {
			t.Error("expected os.ErrNotExist", err)
		}
	})

	t.Run("incorrect format returns err", func(t *testing.T) {
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
	})

	t.Run("unreadable file returns err", func(t *testing.T) {
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
	})
}

func TestWriteConfig(t *testing.T) {
	t.Run("ulid", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "deeply", ConfigFile)

		cfg := Config{UserID: ULID(ulidID)}

		if err := writeConfigToFile(path, cfg); err != nil {
			t.Error("failed to create file", err)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			t.Error("failed to read file", err)
		}

		exp := fmt.Sprintf(`%sanonymous_user_id: %s
`, header, ulidID.String())

		if d := cmp.Diff(exp, string(contents)); d != "" {
			t.Error("contents do not match", d)
		}
	})

	t.Run("uuid", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "deeply", ConfigFile)

		cfg := Config{UserUUID: UUID(uuidID)}

		if err := writeConfigToFile(path, cfg); err != nil {
			t.Error("failed to create file", err)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			t.Error("failed to read file", err)
		}

		exp := fmt.Sprintf(`%sanonymous_user_uuid: %s
`, header, uuidID.String())

		if d := cmp.Diff(exp, string(contents)); d != "" {
			t.Error("contents do not match", d)
		}
	})

	t.Run("ulid and uuid", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "deeply", ConfigFile)

		cfg := Config{
			UserID:   ULID(ulidID),
			UserUUID: UUID(uuidID),
		}

		if err := writeConfigToFile(path, cfg); err != nil {
			t.Error("failed to create file", err)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			t.Error("failed to read file", err)
		}

		exp := fmt.Sprintf(`%sanonymous_user_id: %s
anonymous_user_uuid: %s
`, header, ulidID.String(), uuidID.String())

		if d := cmp.Diff(exp, string(contents)); d != "" {
			t.Error("contents do not match", d)
		}
	})
}

func TestUUID(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		uuid := NewUUID()
		if d := cmp.Diff(36, len(uuid.String())); d != "" {
			t.Error("uuid length mismatch", d)
		}
	})

	t.Run("yaml marshal", func(t *testing.T) {
		uuid := NewUUID()
		s, err := yaml.Marshal(uuid)
		if err != nil {
			t.Error("failed to marshal uuid", err)
		}
		if d := cmp.Diff(uuid.String(), strings.TrimSpace(string(s))); d != "" {
			t.Error("uuid values do not match", d)
		}
	})

	t.Run("yaml unmarshal", func(t *testing.T) {
		var uuid UUID
		if err := yaml.Unmarshal([]byte(NewUUID().String()), &uuid); err != nil {
			t.Error("failed to unmarshal uuid", err)
		}

		if d := cmp.Diff(36, len(uuid.String())); d != "" {
			t.Error("uuid length mismatch", d)
		}
	})

	t.Run("isZero", func(t *testing.T) {
		uuid := UUID(uuid.Nil)
		if d := cmp.Diff(true, uuid.IsZero()); d != "" {
			t.Error("uuid should zero", d)
		}

		uuid = NewUUID()
		if d := cmp.Diff(false, uuid.IsZero()); d != "" {
			t.Error("uuid should zero", d)
		}
	})

	t.Run("toUUID", func(t *testing.T) {
		uuid := NewUUID()
		if d := cmp.Diff(36, len(uuid.toUUID().String())); d != "" {
			t.Error("uuid length mismatch", d)
		}
	})
}
