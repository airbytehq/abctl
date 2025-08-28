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
}

func newSelectModel(prompt string, options []string) SelectModel {
	return SelectModel{
		prompt:  prompt,
		options: options,
	}
}

func (m SelectModel) Init() tea.Cmd {
	return nil
}

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
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SelectModel) View() string {
	s := fmt.Sprintf("\033[1m%s\033[0m\n\n", m.prompt) // Bold text with extra newline

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