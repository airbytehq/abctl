package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FilterableSelectModel implements a bubbletea model for filterable selection with a visible text input
type FilterableSelectModel struct {
	prompt      string
	options     []string
	filtered    []string
	selected    int
	filterInput textinput.Model
	cancelled   bool
	done        bool
}

func newFilterableSelectModel(prompt string, options []string) FilterableSelectModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 60
	ti.Prompt = "> "
	// Purple color for Airbyte branding (#664EFF or close to it)
	purpleColor := lipgloss.Color("99") // Terminal 256 color close to purple
	ti.PromptStyle = lipgloss.NewStyle().Foreground(purpleColor)
	ti.TextStyle = lipgloss.NewStyle().Underline(true).Foreground(purpleColor)
	ti.PlaceholderStyle = lipgloss.NewStyle().Underline(true).Foreground(purpleColor)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(purpleColor)

	// Set initial placeholder to first option
	if len(options) > 0 {
		ti.Placeholder = options[0]
	} else {
		ti.Placeholder = "Start typing to filter..."
	}

	return FilterableSelectModel{
		prompt:      prompt,
		options:     options,
		filtered:    options, // Start with all options visible
		filterInput: ti,
		selected:    0,
	}
}

// Init initializes the filterable select model with cursor blink
func (m FilterableSelectModel) Init() tea.Cmd {
	// Start with text input blinking
	return textinput.Blink
}

// Update handles input events for the filterable select model
func (m FilterableSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 {
				m.done = true
				return m, tea.Quit
			}
		case "up", "ctrl+p":
			if len(m.filtered) > 0 && m.selected > 0 {
				m.selected--
			}
			return m, nil
		case "down", "ctrl+n", "tab":
			if len(m.filtered) > 0 && m.selected < len(m.filtered)-1 {
				m.selected++
			}
			return m, nil
		}
	}

	// Let the textinput handle all other input (including paste)
	var cmd tea.Cmd
	prevValue := m.filterInput.Value()
	m.filterInput, cmd = m.filterInput.Update(msg)

	// If the value changed, re-filter
	if m.filterInput.Value() != prevValue {
		m.filterOptions()
	}

	return m, cmd
}

func (m *FilterableSelectModel) filterOptions() {
	filter := strings.ToLower(m.filterInput.Value())

	if filter == "" {
		m.filtered = m.options
		return
	}

	var filtered []string
	for _, opt := range m.options {
		if strings.Contains(strings.ToLower(opt), filter) {
			filtered = append(filtered, opt)
		}
	}

	m.filtered = filtered

	// Auto-select the first match when filtering
	if len(filtered) > 0 {
		m.selected = 0
		// Update placeholder to show the best match
		m.filterInput.Placeholder = filtered[0]
	}
}

// View renders the filterable select model
func (m FilterableSelectModel) View() string {
	s := fmt.Sprintf("\033[1m%s\033[0m\n\n", m.prompt) // Bold title

	// If done, only show the selected item
	if m.done {
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			s += fmt.Sprintf("> %s\n\n", m.filtered[m.selected])
		}
		return s
	}

	// Show filter input area
	filterValue := m.filterInput.Value()
	if filterValue != "" {
		s += fmt.Sprintf("Filter: %s\n", filterValue)
		s += "────────────────────\n\n"
	} else {
		// Show subtle hint that filtering is available
		s += "\033[2m\033[3mStart typing to filter\033[0m\n\n"
	}

	// Show all filtered options
	if len(m.filtered) == 0 {
		s += "  No matches found\n"
	} else {
		for i, option := range m.filtered {
			if i < 10 { // Show up to 10 items
				cursor := " "
				if i == m.selected {
					cursor = ">"
				}
				s += fmt.Sprintf("%s %s\n", cursor, option)
			}
		}

		// Show count if there are many more
		if len(m.filtered) > 10 {
			s += fmt.Sprintf("\n  \033[2m(%d more...)\033[0m\n", len(m.filtered)-10)
		}
	}

	s += "\nType to filter, press enter to select, esc to cancel"
	return s
}
