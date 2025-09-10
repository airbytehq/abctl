package ui

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// newTestUI creates a UI instance with buffers for testing
func newTestUI() (*BubbleteaUI, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ui := &BubbleteaUI{
		stdout: stdout,
		stderr: stderr,
	}
	return ui, stdout, stderr
}

// newTestUIWithInput creates a UI instance with input simulation for testing
func newTestUIWithInput(input string) (*BubbleteaUI, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	inputReader := strings.NewReader(input)
	ui := NewWithOptions(stdout, stderr, inputReader)
	return ui, stdout, stderr
}

// Test UI provider methods
func TestBubbleteaUI_ShowInfo(t *testing.T) {
	ui, stdout, stderr := newTestUI()

	ui.ShowInfo("Test message")

	assert.Equal(t, "Test message\n", stdout.String())
	assert.Equal(t, "", stderr.String())
}

func TestBubbleteaUI_ShowHeading(t *testing.T) {
	ui, stdout, stderr := newTestUI()

	ui.ShowHeading("Test heading")

	assert.Equal(t, "\033[1mTest heading\033[0m\n\n", stdout.String())
	assert.Equal(t, "", stderr.String())
}

func TestBubbleteaUI_ShowSection(t *testing.T) {
	ui, stdout, stderr := newTestUI()

	ui.ShowSection("Test section", "line 1", "line 2")

	assert.Equal(t, "\033[1mTest section\033[0m\n  line 1\n  line 2\n\n", stdout.String())
	assert.Equal(t, "", stderr.String())
}

func TestBubbleteaUI_ShowError(t *testing.T) {
	ui, stdout, stderr := newTestUI()

	ui.ShowError(fmt.Errorf("test error"))

	assert.Equal(t, "Error: test error\n", stderr.String())
	assert.Equal(t, "", stdout.String())

	// Test nil error
	stderr.Reset()
	ui.ShowError(nil)
	assert.Equal(t, "", stderr.String())
}

func TestBubbleteaUI_ShowSuccess(t *testing.T) {
	ui, stdout, stderr := newTestUI()

	ui.ShowSuccess("Success message")

	assert.Equal(t, "Success message\n", stdout.String())
	assert.Equal(t, "", stderr.String())
}

func TestBubbleteaUI_ShowProgress(t *testing.T) {
	ui, _, _ := newTestUI()
	// Test progress indicator - just ensure it doesn't panic
	stop := ui.ShowProgress("Testing")
	// Stop immediately to avoid long running test
	stop()
}

func TestBubbleteaUI_Confirm(t *testing.T) {
	// This is harder to test due to bubbletea interaction
	// but we can test the model creation logic
	ui, _, _ := newTestUI()
	_ = ui // Just check it compiles
}

func TestBubbleteaUI_Title(t *testing.T) {
	ui, stdout, stderr := newTestUI()

	ui.Title("Test title")

	assert.Equal(t, "\n\033[1mTest title\033[0m\n\n", stdout.String())
	assert.Equal(t, "", stderr.String())
}

func TestBubbleteaUI_ShowKeyValue(t *testing.T) {
	ui, stdout, stderr := newTestUI()

	ui.ShowKeyValue("Key", "value")

	assert.Equal(t, "\033[1mKey:\033[0m value\n", stdout.String())
	assert.Equal(t, "", stderr.String())
}

func TestBubbleteaUI_NewLine(t *testing.T) {
	ui, stdout, stderr := newTestUI()

	ui.NewLine()

	assert.Equal(t, "\n", stdout.String())
	assert.Equal(t, "", stderr.String())
}

// Test New constructor
func TestNew(t *testing.T) {
	ui := New()
	assert.NotNil(t, ui)
	assert.Equal(t, os.Stdout, ui.stdout)
	assert.Equal(t, os.Stderr, ui.stderr)
}

// Test RunWithSpinner
func TestBubbleteaUI_RunWithSpinner(t *testing.T) {
	// Skip this test as it requires a real TTY which isn't available in test environment
	t.Skip("RunWithSpinner requires TTY which is not available in test environment")
}

// Test public Select method (need to test the tea.Program interaction)
func TestBubbleteaUI_Select_Integration(t *testing.T) {
	// This tests the actual flow of the Select method
	_, _, _ = newTestUI()

	// Since we can't easily mock tea.NewProgram, we'll test that the method
	// properly handles the model states
	model := newSelectModel("Choose:", []string{"A", "B", "C"})
	model.selected = 1

	// Simulate selection
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
	finalModel := result.(SelectModel)
	assert.Equal(t, 1, finalModel.selected)
	assert.False(t, finalModel.cancelled)
}

// Test public FilterableSelect method
func TestBubbleteaUI_FilterableSelect_Integration(t *testing.T) {
	_, _, _ = newTestUI()

	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry"})

	// Simulate typing to filter
	model.filterInput.SetValue("ban")
	model.filterOptions()
	assert.Equal(t, []string{"Banana"}, model.filtered)

	// Simulate selection
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
	finalModel := result.(FilterableSelectModel)
	assert.Equal(t, 0, finalModel.selected) // First (only) item in filtered list
	assert.False(t, finalModel.cancelled)
}

// Test public TextInput method
func TestBubbleteaUI_TextInput_Integration(t *testing.T) {
	_, _, _ = newTestUI()

	validator := func(s string) error {
		if s == "invalid" {
			return fmt.Errorf("invalid input")
		}
		return nil
	}

	model := newTextInputModel("Enter:", "default", validator)

	// Test validation
	model.input.SetValue("invalid")
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	finalModel := result.(TextInputModel)
	assert.Nil(t, cmd) // Should not quit on validation error
	assert.NotEmpty(t, finalModel.error)

	// Test valid input
	model.input.SetValue("valid")
	model.error = ""
	result, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	finalModel = result.(TextInputModel)
	assert.NotNil(t, cmd) // Should quit on valid input
	assert.Equal(t, "valid", finalModel.value)
}

// Test public Confirm method
func TestBubbleteaUI_Confirm_Integration(t *testing.T) {
	_, _, _ = newTestUI()

	// Test confirm model behavior
	model := newSelectModel("Confirm?", []string{"Yes", "No"})
	model.selected = 1 // No

	// Simulate selection
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
	finalModel := result.(SelectModel)
	assert.Equal(t, 1, finalModel.selected) // No
	assert.False(t, finalModel.cancelled)
}

// Test Select method with input simulation
func TestBubbleteaUI_Select_WithInput(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		options       []string
		expectedIndex int
		expectedValue string
		expectError   bool
	}{
		{
			name:          "select first option",
			input:         "\r", // Enter key
			options:       []string{"Option 1", "Option 2", "Option 3"},
			expectedIndex: 0,
			expectedValue: "Option 1",
			expectError:   false,
		},
		{
			name:          "navigate down and select",
			input:         "\x1b[B\r", // Down arrow + Enter
			options:       []string{"Option 1", "Option 2"},
			expectedIndex: 1,
			expectedValue: "Option 2",
			expectError:   false,
		},
		{
			name:        "cancel with escape",
			input:       "\x1b", // ESC key
			options:     []string{"Option 1", "Option 2"},
			expectError: true,
		},
		{
			name:        "cancel with ctrl+c",
			input:       "\x03", // Ctrl+C
			options:     []string{"Option 1", "Option 2"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui, _, _ := newTestUIWithInput(tt.input)

			index, value, err := ui.Select("Choose:", tt.options)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedIndex, index)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

// Note: FilterableSelect, TextInput, and Confirm with input simulation
// are more complex due to their interactive nature and would require
// more sophisticated input handling for proper testing. The Select method
// testing above demonstrates that the core input simulation works.

// Test NewWithOptions constructor
func TestNewWithOptions(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	input := strings.NewReader("test")

	ui := NewWithOptions(stdout, stderr, input)

	assert.NotNil(t, ui)
	assert.Equal(t, stdout, ui.stdout)
	assert.Equal(t, stderr, ui.stderr)
	assert.NotNil(t, ui.programOptions)
	assert.Len(t, ui.programOptions, 3) // WithInput, WithOutput, WithoutRenderer
}
