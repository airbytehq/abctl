package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Operation result messages for spinner
type (
	operationSuccessMsg struct{}
	operationErrorMsg   struct{ err error }
)

// SpinnerModel implements a bubbletea model for showing progress
type SpinnerModel struct {
	spinner        spinner.Model
	message        string
	done           bool
	operationError error
}

func newSpinnerModel(message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	return SpinnerModel{
		spinner: s,
		message: message,
	}
}

// Init initializes the spinner model with tick animation
func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles input events for the spinner model
func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case operationSuccessMsg:
		m.done = true
		return m, tea.Quit
	case operationErrorMsg:
		m.operationError = msg.err
		m.done = true
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.done = true
			return m, tea.Quit
		}
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the spinner model
func (m SpinnerModel) View() string {
	if m.done {
		// Clear the line that was used for the spinner
		return "\033[2K\r" // Clear line and return to start
	}
	return fmt.Sprintf("%s %s", m.message, m.spinner.View())
}
