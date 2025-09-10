package ui

import (
	"encoding/json"
	"fmt"
)

// ShowJSON displays formatted JSON output
func (ui *BubbleteaUI) ShowJSON(data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(ui.stdout, string(jsonData))
	return nil
}
