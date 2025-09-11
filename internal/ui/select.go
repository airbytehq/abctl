package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// SelectModel implements a bubbletea model for selecting from a list
type SelectModel struct {
	prompt    string
	options   []string
	selected  int
	cancelled bool
	done      bool
}

func newSelectModel(prompt string, options []string) SelectModel {
	return SelectModel{
		prompt:  prompt,
		options: options,
	}
}

// Init initializes the select model
func (m SelectModel) Init() tea.Cmd {
	return nil
}

// Update handles input events for the select model
func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.options)-1 {
				m.selected++
			}
		case "enter":
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the select model
func (m SelectModel) View() string {
	s := fmt.Sprintf("\033[1m%s\033[0m\n\n", m.prompt) // Bold text with extra newline

	// If done, only show the selected option
	if m.done {
		s += fmt.Sprintf("> %s\n\n", m.options[m.selected])
		return s
	}

	for i, option := range m.options {
		cursor := " "
		if i == m.selected {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, option)
	}

	s += "\nPress enter to select, esc to cancel"
	return s
}
