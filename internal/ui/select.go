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
	viewport  int // Top of viewport for scrolling
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
				m.adjustViewport()
			}
		case "down", "j":
			if m.selected < len(m.options)-1 {
				m.selected++
				m.adjustViewport()
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

// adjustViewport ensures the selected item is visible in the viewport
func (m *SelectModel) adjustViewport() {
	const maxVisible = 10 // Show max 10 items at once
	
	// If selected is above viewport, scroll up
	if m.selected < m.viewport {
		m.viewport = m.selected
	}
	
	// If selected is below viewport, scroll down
	if m.selected >= m.viewport+maxVisible {
		m.viewport = m.selected - maxVisible + 1
	}
}

// View renders the select model
func (m SelectModel) View() string {
	s := fmt.Sprintf("\033[1m%s\033[0m\n\n", m.prompt) // Bold text with extra newline

	// If done, only show the selected option
	if m.done {
		s += fmt.Sprintf("> %s\n\n", m.options[m.selected])
		return s
	}

	const maxVisible = 10
	visibleEnd := m.viewport + maxVisible
	if visibleEnd > len(m.options) {
		visibleEnd = len(m.options)
	}

	// Show only the visible portion of options
	for i := m.viewport; i < visibleEnd; i++ {
		cursor := " "
		if i == m.selected {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, m.options[i])
	}

	s += "\nPress enter to select, esc to cancel"
	return s
}
