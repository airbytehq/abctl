package ui

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Provider provides interactive terminal UI components
type Provider interface {
	// Select presents a list of options and returns the selected index and value
	Select(prompt string, options []string) (int, string, error)

	// FilterableSelect presents a filterable list of options and returns the selected index and value
	FilterableSelect(prompt string, options []string) (int, string, error)

	// TextInput prompts for text input with optional validation
	TextInput(prompt string, defaultValue string, validator func(string) error) (string, error)

	// Confirm asks a yes/no question
	Confirm(prompt string, defaultValue bool) (bool, error)

	// ShowProgress displays a progress indicator with a message
	ShowProgress(message string) func()

	// RunWithSpinner runs a function with a bubbletea spinner
	RunWithSpinner(message string, operation func() error) error

	// ShowInfo displays an informational message (no emoji, no special formatting)
	ShowInfo(message string)

	// Title displays a large title with extra spacing
	Title(message string)

	// ShowHeading displays a heading/section title
	ShowHeading(message string)

	// ShowSection displays a section block with heading and indented content
	ShowSection(heading string, lines ...string)

	// ShowKeyValue displays a key-value pair with bold key
	ShowKeyValue(key, value string)

	// NewLine prints a blank line
	NewLine()

	// ShowError displays an error message
	ShowError(err error)

	// ShowSuccess displays a success message
	ShowSuccess(message string)

	// ShowJSON displays formatted JSON output
	ShowJSON(data any) error

	// ShowYAML displays formatted YAML output
	ShowYAML(data any) error
}

// Option represents a selectable option with additional details
type Option struct {
	Label       string
	Value       string
	Description string
}

// BubbleteaUI implementation of the UI Provider interface.
type BubbleteaUI struct {
	stdout         io.Writer
	stderr         io.Writer
	programOptions []tea.ProgramOption
}

// New creates a new UI instance using bubbletea
func New() *BubbleteaUI {
	return &BubbleteaUI{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// NewWithOptions creates a new UI instance with custom options for testing
func NewWithOptions(stdout, stderr io.Writer, input io.Reader) *BubbleteaUI {
	var options []tea.ProgramOption

	if input != nil {
		options = append(options, tea.WithInput(input))
	}

	if stdout != nil {
		options = append(options, tea.WithOutput(stdout))
	}

	// Always disable renderer for testing to avoid TTY issues
	options = append(options, tea.WithoutRenderer())

	return &BubbleteaUI{
		stdout:         stdout,
		stderr:         stderr,
		programOptions: options,
	}
}

// Select presents a list of options and returns the selected index and value
func (ui *BubbleteaUI) Select(prompt string, options []string) (int, string, error) {
	model := newSelectModel(prompt, options)
	program := tea.NewProgram(model, ui.programOptions...)

	finalModel, err := program.Run()
	if err != nil {
		return 0, "", fmt.Errorf("error running select: %w", err)
	}

	selectModel := finalModel.(SelectModel)
	if selectModel.cancelled {
		return 0, "", fmt.Errorf("selection cancelled")
	}

	return selectModel.selected, options[selectModel.selected], nil
}

// FilterableSelect presents a filterable list of options and returns the selected index and value
func (ui *BubbleteaUI) FilterableSelect(prompt string, options []string) (int, string, error) {
	model := newFilterableSelectModel(prompt, options)
	program := tea.NewProgram(model, ui.programOptions...)

	finalModel, err := program.Run()
	if err != nil {
		return 0, "", fmt.Errorf("error running filterable select: %w", err)
	}

	m := finalModel.(FilterableSelectModel)
	if m.cancelled {
		return 0, "", fmt.Errorf("selection cancelled")
	}

	// Get the selected value from filtered list
	if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
		return 0, "", fmt.Errorf("no valid selection")
	}

	// Find the original index of the selected filtered item
	selectedValue := m.filtered[m.selected]
	for i, option := range options {
		if option == selectedValue {
			return i, selectedValue, nil
		}
	}

	return 0, "", fmt.Errorf("selected item not found in original options")
}

// TextInput prompts for text input with optional validation
func (ui *BubbleteaUI) TextInput(prompt string, defaultValue string, validator func(string) error) (string, error) {
	model := newTextInputModel(prompt, defaultValue, validator)
	program := tea.NewProgram(model, ui.programOptions...)

	finalModel, err := program.Run()
	if err != nil {
		return "", fmt.Errorf("error running text input: %w", err)
	}

	textModel := finalModel.(TextInputModel)
	if textModel.cancelled {
		return "", fmt.Errorf("input cancelled")
	}

	return textModel.value, nil
}

// Confirm asks a yes/no question and returns the user's choice
func (ui *BubbleteaUI) Confirm(prompt string, defaultValue bool) (bool, error) {
	options := []string{"Yes", "No"}
	defaultIndex := 0
	if !defaultValue {
		defaultIndex = 1
	}

	model := newSelectModel(prompt, options)
	model.selected = defaultIndex
	program := tea.NewProgram(model, ui.programOptions...)

	finalModel, err := program.Run()
	if err != nil {
		return false, fmt.Errorf("error running confirm: %w", err)
	}

	selectModel := finalModel.(SelectModel)
	if selectModel.cancelled {
		return false, fmt.Errorf("confirmation cancelled")
	}

	return selectModel.selected == 0, nil
}

// ShowProgress displays a progress indicator (spinning dots) and returns a function to stop it
func (ui *BubbleteaUI) ShowProgress(message string) func() {
	fmt.Fprintf(ui.stdout, "%s ", message)
	return func() {
		fmt.Fprintln(ui.stdout, " ✓")
	}
}

// RunWithSpinner runs a function with a bubbletea spinner
func (ui *BubbleteaUI) RunWithSpinner(message string, operation func() error) error {
	model := newSpinnerModel(message)
	program := tea.NewProgram(model, ui.programOptions...)

	// Run the operation in a goroutine
	go func() {
		err := operation()
		if err != nil {
			program.Send(operationErrorMsg{err})
		} else {
			program.Send(operationSuccessMsg{})
		}
	}()

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("spinner error: %w", err)
	}

	spinnerModel := finalModel.(SpinnerModel)
	return spinnerModel.operationError
}

// ShowInfo displays an informational message (no emoji, no special formatting)
func (ui *BubbleteaUI) ShowInfo(message string) {
	fmt.Fprintln(ui.stdout, message)
}

// Title displays a large title with extra spacing
func (ui *BubbleteaUI) Title(message string) {
	fmt.Fprintf(ui.stdout, "\n\033[1m%s\033[0m\n\n", message) // Bold text with extra spacing
}

// ShowHeading displays a heading/section title
func (ui *BubbleteaUI) ShowHeading(message string) {
	fmt.Fprintf(ui.stdout, "\033[1m%s\033[0m\n\n", message) // Bold text
}

// ShowError prints an error message to stderr
func (ui *BubbleteaUI) ShowError(err error) {
	if err != nil {
		fmt.Fprintf(ui.stderr, "Error: %s\n", err.Error())
	}
}

// ShowSuccess displays a success message
func (ui *BubbleteaUI) ShowSuccess(message string) {
	fmt.Fprintln(ui.stdout, message)
}

// ShowSection displays a section block with heading and indented content
func (ui *BubbleteaUI) ShowSection(heading string, lines ...string) {
	fmt.Fprintf(ui.stdout, "\033[1m%s\033[0m\n", heading) // Bold text
	for _, line := range lines {
		fmt.Fprintf(ui.stdout, "  %s\n", line)
	}
	fmt.Fprintln(ui.stdout)
}

// ShowKeyValue displays a key-value pair with bold key
func (ui *BubbleteaUI) ShowKeyValue(key, value string) {
	fmt.Fprintf(ui.stdout, "\033[1m%s:\033[0m %s\n", key, value)
}

// NewLine prints a blank line
func (ui *BubbleteaUI) NewLine() {
	fmt.Fprintln(ui.stdout)
}
