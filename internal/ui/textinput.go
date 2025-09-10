package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// TextInputModel implements a bubbletea model for text input
type TextInputModel struct {
	prompt      string
	value       string
	placeholder string
	input       textinput.Model
	validator   func(string) error
	cancelled   bool
	error       string
	done        bool
}

func newTextInputModel(prompt string, defaultValue string, validator func(string) error) TextInputModel {
	ti := textinput.New()
	ti.Focus()
	ti.Placeholder = defaultValue
	ti.CharLimit = 200
	ti.Width = 60

	return TextInputModel{
		prompt:      prompt,
		placeholder: defaultValue,
		input:       ti,
		validator:   validator,
	}
}

// Init initializes the text input model with cursor blink
func (m TextInputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles input events for the text input model
func (m TextInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// First let the textinput handle the message to support paste
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	// Keep our value in sync with the textinput
	m.value = m.input.Value()

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
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		default:
			m.error = "" // Clear error on input change
		}
	}

	return m, cmd
}

// View renders the text input model
func (m TextInputModel) View() string {
	s := fmt.Sprintf("\033[1m%s\033[0m\n\n", m.prompt)

	// If done, only show the final value
	if m.done {
		s += fmt.Sprintf("> %s\n\n", m.value)
		return s
	}

	s += fmt.Sprintf("%s\n", m.input.View())

	if m.error != "" {
		s += fmt.Sprintf("\n\033[31mError: %s\033[0m\n", m.error)
		s += "Press enter to retry or esc to cancel\n"
	} else {
		s += "\nPress enter to confirm, esc to cancel\n"
	}

	return s
}
