package cmd

import (
	"fmt"
	"strings"

	"github.com/airbytehq/abctl/internal/ui"
)

// outputHandler defines how to handle a specific output format
type outputHandler func(ui.Provider, any) error

// outputHandlers maps format names to their handlers
var outputHandlers = map[string]outputHandler{
	"json": func(ui ui.Provider, data any) error {
		return ui.ShowJSON(data)
	},
	"yaml": func(ui ui.Provider, data any) error {
		return ui.ShowYAML(data)
	},
}

// RenderOutput displays data in the specified format using the UI provider
// Defaults to JSON if format is empty
func RenderOutput(uiProvider ui.Provider, data any, format string) error {
	// Default to JSON
	if format == "" {
		format = "json"
	}

	handler, exists := outputHandlers[format]
	if !exists {
		supported := make([]string, 0, len(outputHandlers))
		for k := range outputHandlers {
			supported = append(supported, k)
		}
		return fmt.Errorf("unsupported output format: %s (supported: %s)", format, strings.Join(supported, ", "))
	}

	return handler(uiProvider, data)
}