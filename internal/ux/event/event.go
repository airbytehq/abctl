package event

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"sync"
)

type MsgType int8

const (
	DEBUG MsgType = iota
	INFO
	WARN
	ERROR
)

type Msg struct {
	Type MsgType
	Msg  string
}

var _ tea.Model = (*Model)(nil)

type Model struct{}

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
	}

	return m, nil
}

func (m Model) View() string {
	return ""
}

var leftPad = strings.Repeat(" ", 5)
var mutex sync.Mutex

func handleLogMsg(msg Msg) tea.Cmd {
	msg.Msg = strings.ReplaceAll(msg.Msg, "\n", "\n"+leftPad)

	var prefix string
	switch msg.Type {
	case DEBUG:
		prefix = fmtDebug.String()
		//s = fmt.Sprintf("%s %s", fmtDebug, msg.Msg)
	case INFO:
		prefix = fmtInfo.String()
		//s = fmt.Sprintf("%s %s", fmtInfo, msg.Msg)
	case WARN:
		prefix = fmtWarn.String()
		//s = fmt.Sprintf("%s %s", fmtWarn, msg.Msg)
	case ERROR:
		prefix = fmtErr.String()
		//s = fmt.Sprintf("%s %s", fmtErr, msg.Msg)
	}

	return tea.Println(prefix + " " + msg.Msg)
}

var (
	fmtDebug = logStyle("debug", "63")
	fmtWarn  = logStyle("warning", "192")
	fmtInfo  = logStyle("info", "86")
	fmtErr   = logStyle("error", "204")
)

func logStyle(prefix, color string) lipgloss.Style {
	switch {
	case len(prefix) < 4:
		prefix += strings.Repeat(" ", 4-len(prefix))
	case len(prefix) > 4:
		prefix = prefix[:4]
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		SetString(strings.ToUpper(prefix))
	//Bold(true)
}
