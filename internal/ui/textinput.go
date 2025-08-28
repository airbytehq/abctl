package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// TextInputModel implements a bubbletea model for text input
type TextInputModel struct {
	prompt      string
	value       string
	placeholder string
	validator   func(string) error
	cancelled   bool
	error       string
}

func newTextInputModel(prompt string, defaultValue string, validator func(string) error) TextInputModel {
	return TextInputModel{
		prompt:      prompt,
		value:       "",
		placeholder: defaultValue,
		validator:   validator,
	}
}

func (m TextInputModel) Init() tea.Cmd {
	return nil
}

func (m TextInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Use placeholder if value is empty
			if m.value == "" && m.placeholder != "" {
				m.value = m.placeholder
			}

			if m.validator != nil {
				if err := m.validator(m.value); err != nil {
					m.error = err.Error()
					return m, nil
				}
			}
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "backspace":
			if len(m.value) > 0 {
				m.value = m.value[:len(m.value)-1]
				m.error = "" // Clear error on input change
			}
		default:
			// Add character to value
			if len(msg.String()) == 1 {
				m.value += msg.String()
				m.error = "" // Clear error on input change
			}
		}
	}
	return m, nil
}

func (m TextInputModel) View() string {
	var displayValue string
	if m.value == "" && m.placeholder != "" {
		// Make placeholder italic and slightly dimmed
		displayValue = fmt.Sprintf("\033[3m\033[2m%s\033[0m", m.placeholder)
	} else {
		displayValue = m.value
	}
	s := fmt.Sprintf("%s\n> %s", m.prompt, displayValue)

	if m.error != "" {
		s += fmt.Sprintf("\n❌ %s", m.error)
	}

	s += "\n\nPress enter to confirm, esc to cancel"
	return s
}