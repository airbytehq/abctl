package ui

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestSpinnerModel(t *testing.T) {
	model := newSpinnerModel("Loading...")

	// Test Init
	cmd := model.Init()
	assert.NotNil(t, cmd) // Should start the spinner

	// Test View
	view := model.View()
	assert.Contains(t, view, "Loading...")

	// Test Update with success message
	result, cmd := model.Update(operationSuccessMsg{})
	model = result.(SpinnerModel)
	assert.NotNil(t, cmd) // Should quit
	assert.True(t, model.done)

	// Test Update with tick
	model.done = false
	result, cmd = model.Update(spinner.TickMsg{})
	model = result.(SpinnerModel)
	assert.NotNil(t, cmd) // Should continue ticking

	// Test Update with error message
	model.done = false
	result, cmd = model.Update(operationErrorMsg{err: fmt.Errorf("test error")})
	model = result.(SpinnerModel)
	assert.NotNil(t, cmd) // Should quit
	assert.Equal(t, fmt.Errorf("test error"), model.operationError)
	assert.True(t, model.done)
}

func TestSpinnerModel_ViewDone(t *testing.T) {
	model := newSpinnerModel("Loading...")
	model.done = true

	view := model.View()
	assert.Equal(t, "\033[2K\r", view) // Should return clear line
}

func TestSpinnerModel_UpdateMoreCases(t *testing.T) {
	model := newSpinnerModel("Loading...")

	// Test with Ctrl+C
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model = result.(SpinnerModel)
	assert.NotNil(t, cmd) // Should quit
	assert.True(t, model.done)
}
