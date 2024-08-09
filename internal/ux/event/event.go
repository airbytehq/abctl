package event

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

type MsgType int8

const (
	DEBUG MsgType = iota
	INFO
	WARN
	ERROR
)

func Debug(msg string) Msg {
	return Msg{Type: DEBUG, Msg: msg}
}

func Info(msg string) Msg {
	return Msg{Type: INFO, Msg: msg}
}

func Warn(msg string) Msg {
	return Msg{Type: WARN, Msg: msg}
}

func Error(msg string) Msg {
	return Msg{Type: ERROR, Msg: msg}
}

type Msg struct {
	Type MsgType
	Msg  string
}

var _ tea.Model = (*Model)(nil)

type Model struct {
	err error
}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case Msg:
		return m, handleLogMsg(msg)
	case error:
		m.err = msg
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	return ""
}

var leftPad = strings.Repeat(" ", 5)

func handleLogMsg(msg Msg) tea.Cmd {
	msg.Msg = strings.ReplaceAll(msg.Msg, "\n", "\n"+leftPad)

	var prefix string
	switch msg.Type {
	case DEBUG:
		prefix = fmtDebug.Render("")
	case INFO:
		prefix = "" //fmtInfo.String()
	case WARN:
		prefix = fmtWarn.Render("")
	case ERROR:
		prefix = fmtErr.Render("")
	}

	return tea.Println(prefix + msg.Msg)
}

var (
	fmtDebug = logStyle("debug", "63", "63")
	fmtInfo  = logStyle("info", "39", "86") // 79
	fmtWarn  = logStyle("warning", "208", "192")
	fmtErr   = logStyle("error", "203", "204")
)

func logStyle(prefix, colorLight, colorDark string) lipgloss.Style {
	switch {
	case len(prefix) < 4:
		prefix += strings.Repeat(" ", 4-len(prefix))
	case len(prefix) > 4:
		prefix = prefix[:4]
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: colorLight, Dark: colorDark}).
		SetString(strings.ToUpper(prefix))
}
