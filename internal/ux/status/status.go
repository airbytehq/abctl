package status

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"time"
)

type MsgType int

const (
	UPDATE MsgType = iota
	SUCCESS
	FAILURE
)

type Msg struct {
	Type MsgType
	Msg  string
}

var _ tea.Model = (*Model)(nil)

type Model struct {
	spinner spinner.Model
	start   time.Time
	msg     string
	running bool
}

func New(msg string) Model {
	s := spinner.New()
	//s.Model = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		spinner: s,
		start:   time.Now(),
		msg:     msg,
		running: true,
	}
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case Msg:
		return m, handleMsg(&m, msg)
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return fmt.Sprintf("%s %s (%s)", m.spinner.View(), m.msg, formatTime(m.start))
}

func handleMsg(s *Model, msg Msg) tea.Cmd {
	switch msg.Type {
	case UPDATE:
		s.msg = msg.Msg
	case SUCCESS:
		return tea.Println(fmt.Sprintf("%s %s", fmtSuccess, msg.Msg))
	case FAILURE:
		return tea.Println(fmt.Sprintf("%s %s", fmtFailure, msg.Msg))
	}

	return nil
}

func formatTime(start time.Time) string {
	return time.Now().Sub(start).Truncate(time.Second).String()
}

var (
	fmtSuccess = spinnerMsgStyle("✓", "70")
	fmtFailure = spinnerMsgStyle("x", "204")
)

func spinnerMsgStyle(prefix, color string) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Bold(true).
		SetString(prefix)
}
