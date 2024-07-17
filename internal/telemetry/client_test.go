package telemetry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	instance = nil
	home := t.TempDir()

	cli := Get(WithUserHome(home))
	if _, ok := cli.(*SegmentClient); !ok {
		t.Error(fmt.Sprintf("expected SegmentClient; received: %T", cli))
	}

	// verify configuration file was created
	data, err := os.ReadFile(filepath.Join(home, ConfigFile))
	if err != nil {
		t.Error("reading config file", err)
	}

	// and has some data
	if !strings.Contains(string(data), "Airbyte") {
		t.Error("expected config file to contain 'Airbyte'")
	}

	if !strings.Contains(string(data), fieldAnalyticsID) {
		t.Error(fmt.Sprintf("expected config file to contain '%s'", fieldAnalyticsID))
	}

	if strings.Contains(string(data), fieldUserID) {
		t.Error(fmt.Sprintf("config file should not contain '%s'", fieldUserID))
	}
}

func TestGet_WithExistingULID(t *testing.T) {
	instance = nil
	home := t.TempDir()

	// write a config with a ulid only
	cfg := Config{UserID: NewULID()}
	if err := writeConfigToFile(filepath.Join(home, ConfigFile), cfg); err != nil {
		t.Fatal("failed writing config", err)
	}

	cli := Get(WithUserHome(home))
	if _, ok := cli.(*SegmentClient); !ok {
		t.Error(fmt.Sprintf("expected SegmentClient; received: %T", cli))
	}

	// verify configuration file was created
	data, err := os.ReadFile(filepath.Join(home, ConfigFile))
	if err != nil {
		t.Error("reading config file", err)
	}

	// and has some data
	if !strings.Contains(string(data), "Airbyte") {
		t.Error("expected config file to contain 'Airbyte'")
	}

	if !strings.Contains(string(data), fieldAnalyticsID) {
		t.Error(fmt.Sprintf("expected config file to contain '%s'", fieldAnalyticsID))
	}

	if !strings.Contains(string(data), fieldUserID) {
		t.Error(fmt.Sprintf("config file should not contain '%s'", fieldUserID))
	}
}

func TestGet_SameInstance(t *testing.T) {
	instance = nil
	home := t.TempDir()
	cli1 := Get(WithUserHome(home))
	cli2 := Get(WithUserHome(home))
	cli3 := Get()
	cli4 := Get(WithDNT())

	if cli1 != cli2 {
		t.Error("expected same client")
	}
	if cli1 != cli3 {
		t.Error("expected same client")
	}
	if cli1 != cli4 {
		t.Error("expected same client")
	}
}

func TestGet_Dnt(t *testing.T) {
	instance = nil
	home := t.TempDir()
	cli := Get(WithUserHome(home), WithDNT())

	if _, ok := cli.(NoopClient); !ok {
		t.Error(fmt.Sprintf("expected NoopClient; received: %T", cli))
	}

	// no configuration file was created
	_, err := os.ReadFile(filepath.Join(home, ConfigFile))
	if !errors.Is(err, os.ErrNotExist) {
		t.Error("expected file not exists", err)
	}
}
