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
			t.Fatal("unable to create temp file", err)
		}
		defer f.Close()

		if _, err := f.WriteString(`# comments
anonymous_user_id: ` + ulidID.String()); err != nil {
			t.Fatal("unable to write to temp file", err)
		}

		cfg, err := loadConfigFromFile(f.Name())
		if d := cmp.Diff(nil, err); d != "" {
			t.Error("failed to load file", d)
		}

		if d := cmp.Diff(ulidID.String(), cfg.UserID.String()); d != "" {
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
			t.Fatal("unable to create temp file", err)
		}
		defer f.Close()

		if _, err := f.WriteString(`This is a regular file, not a configuration file!`); err != nil {
			t.Fatal("unable to write to temp file", err)
		}

		_, err = loadConfigFromFile(f.Name())
		if err == nil {
			t.Error("expected an error to be returned")
		}
	})

	t.Run("unreadable file returns err", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "analytics-")
		if err != nil {
			t.Fatal("unable to create temp file", err)
		}
		defer f.Close()

		// remove read permissions from file
		if err := f.Chmod(0222); err != nil {
			t.Fatal("unable to chmod temp file", err)
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
			t.Fatal("unable to create temp file", err)
		}
		defer f.Close()

		cfgData := fmt.Sprintf(`# comments
%s: %s`, fieldAnalyticsID, uuidID.String())

		if _, err := f.WriteString(cfgData); err != nil {
			t.Fatal("unable to write to temp file", err)
		}

		cfg, err := loadConfigFromFile(f.Name())
		if d := cmp.Diff(nil, err); d != "" {
			t.Error("failed to load file", d)
		}

		if d := cmp.Diff(uuidID.String(), cfg.AnalyticsID.String()); d != "" {
			t.Error("id is incorrect", d)
		}
	})

	t.Run("happy path with extra fields", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "analytics-")
		if err != nil {
			t.Fatal("unable to create temp file", err)
		}
		defer f.Close()

		cfgData := fmt.Sprintf(`# comments
%s: %s
extra_field: extra_value
another_field: false
total: 300`,
			fieldAnalyticsID, uuidID.String())

		if _, err := f.WriteString(cfgData); err != nil {
			t.Fatal("unable to write to temp file", err)
		}

		cfg, err := loadConfigFromFile(f.Name())
		if d := cmp.Diff(nil, err); d != "" {
			t.Error("failed to load file", d)
		}

		if d := cmp.Diff(uuidID.String(), cfg.AnalyticsID.String()); d != "" {
			t.Error("id is incorrect", d)
		}

		if d := cmp.Diff("extra_value", cfg.Other["extra_field"]); d != "" {
			t.Error("extra_field is incorrect", d)
		}

		if d := cmp.Diff(false, cfg.Other["another_field"]); d != "" {
			t.Error("another_field is incorrect", d)
		}

		if d := cmp.Diff(300, cfg.Other["total"]); d != "" {
			t.Error("total is incorrect", d)
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
			t.Fatal("unable to create temp file", err)
		}
		defer f.Close()

		if _, err := f.WriteString(`This is a regular file, not a configuration file!`); err != nil {
			t.Fatal("unable to write to temp file", err)
		}

		_, err = loadConfigFromFile(f.Name())
		if err == nil {
			t.Error("expected an error to be returned")
		}
	})

	t.Run("unreadable file returns err", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "analytics-")
		if err != nil {
			t.Fatal("unable to create temp file", err)
		}
		defer f.Close()

		// remove read permissions from file
		if err := f.Chmod(0222); err != nil {
			t.Fatal("unable to chmod temp file", err)
		}

		_, err = loadConfigFromFile(f.Name())
		if err == nil {
			t.Error("expected an error to be returned")
		}
	})
}

func TestLoadConfigWithNilFields(t *testing.T) {
	t.Run("nil anonymous_user_id", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "analytics-")
		if err != nil {
			t.Fatal("unable to create temp file", err)
		}
		defer f.Close()

		cfgData := fmt.Sprintf(`# comments
%s: %s
%s:
`,
			fieldAnalyticsID, uuidID.String(), fieldUserID)

		if _, err := f.WriteString(cfgData); err != nil {
			t.Fatal("unable to write to temp file", err)
		}

		cfg, err := loadConfigFromFile(f.Name())
		if d := cmp.Diff(nil, err); d != "" {
			t.Error("failed to load file", d)
		}

		if d := cmp.Diff(uuidID.String(), cfg.AnalyticsID.String()); d != "" {
			t.Error("analyticsID is incorrect", d)
		}

		if d := cmp.Diff(true, cfg.UserID.IsZero()); d != "" {
			t.Error("userID is incorrect", d)
		}
	})

	t.Run("nil analytics_id", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "analytics-")
		if err != nil {
			t.Fatal("unable to create temp file", err)
		}
		defer f.Close()

		cfgData := fmt.Sprintf(`# comments
%s:
`,
			fieldAnalyticsID)

		if _, err := f.WriteString(cfgData); err != nil {
			t.Fatal("unable to write to temp file", err)
		}

		cfg, err := loadConfigFromFile(f.Name())
		if d := cmp.Diff(nil, err); d != "" {
			t.Error("failed to load file", d)
		}

		if d := cmp.Diff(true, cfg.AnalyticsID.IsZero()); d != "" {
			t.Error("id is incorrect", d)
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

		exp := fmt.Sprintf(`%s%s: %s
`, header, fieldUserID, ulidID.String())

		if d := cmp.Diff(exp, string(contents)); d != "" {
			t.Error("contents do not match", d)
		}
	})

	t.Run("uuid", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "deeply", ConfigFile)

		cfg := Config{AnalyticsID: UUID(uuidID)}

		if err := writeConfigToFile(path, cfg); err != nil {
			t.Error("failed to create file", err)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			t.Error("failed to read file", err)
		}

		exp := fmt.Sprintf(`%s%s: %s
`, header, fieldAnalyticsID, uuidID.String())

		if d := cmp.Diff(exp, string(contents)); d != "" {
			t.Error("contents do not match", d)
		}
	})

	t.Run("uuid and other", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "deeply", ConfigFile)

		cfg := Config{
			AnalyticsID: UUID(uuidID),
			Other: map[string]interface{}{
				"another_field": "another_value",
			},
		}

		if err := writeConfigToFile(path, cfg); err != nil {
			t.Error("failed to create file", err)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			t.Error("failed to read file", err)
		}

		exp := fmt.Sprintf(`%s%s: %s
another_field: another_value
`, header, fieldAnalyticsID, uuidID.String())

		if d := cmp.Diff(exp, string(contents)); d != "" {
			t.Error("contents do not match", d)
		}
	})

	t.Run("ulid and uuid", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "deeply", ConfigFile)

		cfg := Config{
			UserID:      ULID(ulidID),
			AnalyticsID: UUID(uuidID),
		}

		if err := writeConfigToFile(path, cfg); err != nil {
			t.Error("failed to create file", err)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			t.Error("failed to read file", err)
		}

		exp := fmt.Sprintf(`%s%s: %s
%s: %s
`, header, fieldUserID, ulidID.String(), fieldAnalyticsID, uuidID.String())

		if d := cmp.Diff(exp, string(contents)); d != "" {
			t.Error("contents do not match", d)
		}
	})

	t.Run("ulid, uuid, and other", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ConfigFile)

		cfg := Config{
			UserID:      ULID(ulidID),
			AnalyticsID: UUID(uuidID),
			Other: map[string]interface{}{
				"field": "value is here",
				"count": 100,
			},
		}

		if err := writeConfigToFile(path, cfg); err != nil {
			t.Error("failed to create file", err)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			t.Error("failed to read file", err)
		}

		exp := fmt.Sprintf(`%sanonymous_user_id: %s
%s: %s
count: 100
field: value is here
`, header, ulidID.String(), fieldAnalyticsID, uuidID.String())

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
