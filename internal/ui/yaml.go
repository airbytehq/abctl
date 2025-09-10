package ui

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ShowYAML displays formatted YAML output
func (ui *BubbleteaUI) ShowYAML(data any) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Fprint(ui.stdout, string(yamlData))
	return nil
}
