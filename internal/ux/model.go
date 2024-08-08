package ux

import (
	"github.com/airbytehq/abctl/internal/ux/event"
	"github.com/airbytehq/abctl/internal/ux/status"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	status  status.Model
	event   event.Model
	err     error
	running bool
}

var _ tea.Model = (*Model)(nil)

func New() Model {
	s := status.New("Starting message")

	return Model{
		status:  s,
		event:   event.Model{},
		running: true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.status.Init(), m.event.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

	case event.Msg:
		model, cmd := m.event.Update(msg)
		m.event = model.(event.Model)
		return m, tea.Sequence(cmd)

	case spinner.TickMsg, status.Msg:
		// always call the status update
		model, cmd := m.status.Update(msg)
		m.status = model.(status.Model)
		return m, tea.Sequence(cmd)
	}

	return m, nil
}

func (m Model) View() string {
	return m.status.View()
}
