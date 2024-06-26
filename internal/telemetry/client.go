package telemetry

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type EventState string

const (
	Start   EventState = "started"
	Failed  EventState = "failed"
	Success EventState = "succeeded"
)

type EventType string

const (
	Install   EventType = "install"
	Status    EventType = "status"
	Uninstall EventType = "uninstall"
)

// Client interface for telemetry data.
type Client interface {
	// Start should be called as soon quickly as possible.
	Start(context.Context, EventType) error
	// Success should be called only if the activity succeeded.
	Success(context.Context, EventType) error
	// Failure should be called only if the activity failed.
	Failure(context.Context, EventType, error) error
	// Attr should be called to add additional attributes to this activity.
	Attr(key, val string)
	// User returns the user identifier being used by this client
	User() uuid.UUID
}

type getConfig struct {
	dnt      bool
	userHome string
	h        *http.Client
}

// GetOption is for optional configuration of the Get call.
type GetOption func(*getConfig)

// WithDnt tells the Get call to enable the do-not-track configuration.
func WithDnt() GetOption {
	return func(gc *getConfig) {
		gc.dnt = true
	}
}

// WithUserHome tells the Get call which directory should be considered the user's home.
// Primary for testing purposes.
func WithUserHome(userHome string) GetOption {
	return func(gc *getConfig) {
		gc.userHome = userHome
	}
}

var (
	// instance is the configured Client holder
	instance Client
	// lock is to ensure that Get is thread-safe
	lock sync.Mutex
)

// Get returns the already configured telemetry Client or configures a new one returning it.
// If a previously configured Client exists, that one will be returned.
func Get(opts ...GetOption) Client {
	lock.Lock()
	defer lock.Unlock()

	if instance != nil {
		return instance
	}

	var getCfg getConfig
	for _, opt := range opts {
		opt(&getCfg)
	}

	if getCfg.dnt {
		instance = NoopClient{}
		return instance
	}

	if getCfg.userHome == "" {
		getCfg.userHome, _ = os.UserHomeDir()
	}

	getOrCreateConfigFile := func(getCfg getConfig) (Config, error) {
		configPath := filepath.Join(getCfg.userHome, ConfigFile)

		// if no file exists, create a new one
		analyticsCfg, err := loadConfigFromFile(configPath)
		if errors.Is(err, os.ErrNotExist) {
			// file not found, create a new one
			analyticsCfg = Config{AnalyticsID: NewUUID()}
			if err := writeConfigToFile(configPath, analyticsCfg); err != nil {
				return analyticsCfg, fmt.Errorf("could not write file to %s: %w", configPath, err)
			}
			pterm.Info.Println(Welcome)
		} else if err != nil {
			return Config{}, fmt.Errorf("could not load config from %s: %w", configPath, err)
		}

		// if a file exists but doesn't have a uuid, create a new uuid
		if analyticsCfg.AnalyticsID.IsZero() {
			analyticsCfg.AnalyticsID = NewUUID()
			if err := writeConfigToFile(configPath, analyticsCfg); err != nil {
				return analyticsCfg, fmt.Errorf("could not write file to %s: %w", configPath, err)
			}
		}

		return analyticsCfg, nil
	}

	cfg, err := getOrCreateConfigFile(getCfg)
	if err != nil {
		pterm.Warning.Printfln("could not create telemetry config file: %s", err.Error())
		instance = NoopClient{}
	} else {
		instance = NewSegmentClient(cfg)
	}

	return instance
}
