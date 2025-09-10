package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestTextInputModel_Init(t *testing.T) {
	model := newTextInputModel("Enter name:", "default", nil)

	cmd := model.Init()
	assert.NotNil(t, cmd) // textinput returns blink command
	assert.Equal(t, "", model.value)
	assert.Equal(t, "default", model.placeholder)
}

func TestTextInputModel_View(t *testing.T) {
	model := newTextInputModel("Enter name:", "test", nil)

	view := model.View()

	// Should contain prompt
	if !strings.Contains(view, "Enter name:") {
		t.Error("View should contain prompt")
	}

	// Should contain value
	if !strings.Contains(view, "test") {
		t.Error("View should contain current value")
	}

	// Should contain instructions
	if !strings.Contains(view, "Press enter to confirm") {
		t.Error("View should contain instructions")
	}
}

func TestTextInputModel_TextInput(t *testing.T) {
	model := newTextInputModel("Enter:", "", nil)

	// Test typing valid characters
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = result.(TextInputModel)
	if model.value != "a" {
		t.Errorf("Expected value='a', got %s", model.value)
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	model = result.(TextInputModel)
	if model.value != "ab" {
		t.Errorf("Expected value='ab', got %s", model.value)
	}

	// Test hyphen
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	model = result.(TextInputModel)
	if model.value != "ab-" {
		t.Errorf("Expected value='ab-', got %s", model.value)
	}

	// Test number
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	model = result.(TextInputModel)
	if model.value != "ab-1" {
		t.Errorf("Expected value='ab-1', got %s", model.value)
	}
}

func TestTextInputModel_AllCharacters(t *testing.T) {
	model := newTextInputModel("Enter:", "", nil)

	// Test all characters are accepted
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}}) // uppercase
	model = result.(TextInputModel)
	if model.value != "A" {
		t.Errorf("Expected uppercase A to be accepted, got %s", model.value)
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}) // space
	model = result.(TextInputModel)
	if model.value != "A " {
		t.Errorf("Expected space to be accepted, got %s", model.value)
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}}) // special char
	model = result.(TextInputModel)
	if model.value != "A @" {
		t.Errorf("Expected @ to be accepted, got %s", model.value)
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}}) // colon for URLs
	model = result.(TextInputModel)
	if model.value != "A @:" {
		t.Errorf("Expected colon to be accepted, got %s", model.value)
	}
}

func TestTextInputModel_Backspace(t *testing.T) {
	model := newTextInputModel("Enter:", "test", nil)
	model.input.SetValue("test") // Set value manually for backspace test

	// Test backspace
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = result.(TextInputModel)
	assert.Equal(t, "tes", model.input.Value()) // bubbles textinput handles the value

	// Test backspace on empty string
	model.input.SetValue("")
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = result.(TextInputModel)
	if model.input.Value() != "" {
		t.Errorf("Expected value='' after backspace on empty, got %s", model.input.Value())
	}
}

func TestTextInputModel_Validation(t *testing.T) {
	validator := func(input string) error {
		if len(input) < 3 {
			return fmt.Errorf("too short")
		}
		return nil
	}

	model := newTextInputModel("Enter:", "ab", validator)

	// Test enter with invalid input
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(TextInputModel)

	assert.Nil(t, cmd)
	assert.Equal(t, "too short", model.error)

	// Test with valid input - need to clear error by typing
	model.error = "" // Clear previous error
	model.input.SetValue("abc")
	result, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(TextInputModel)

	assert.NotNil(t, cmd)
	assert.Equal(t, "", model.error)
}

func TestTextInputModel_EmptyInputUsesPlaceholder(t *testing.T) {
	model := newTextInputModel("Enter:", "my-dataplane", nil)
	model.input.SetValue("") // Clear the value to test placeholder usage

	// Test enter with empty input uses placeholder
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(TextInputModel)

	assert.Equal(t, "my-dataplane", model.value)
	assert.NotNil(t, cmd)
}

func TestTextInputModel_Cancel(t *testing.T) {
	model := newTextInputModel("Enter:", "", nil)

	// Test escape key
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = result.(TextInputModel)

	assert.NotNil(t, cmd)
	assert.True(t, model.cancelled)
}

func TestTextInputModel_ViewWithError(t *testing.T) {
	model := newTextInputModel("Enter:", "default", nil)
	model.error = "validation error"

	view := model.View()
	assert.Contains(t, view, "validation error")
	assert.Contains(t, view, "Enter:")
}

func TestTextInputModel_ViewWithPlaceholder(t *testing.T) {
	model := newTextInputModel("Enter:", "default", nil)
	model.value = "" // Empty value should show placeholder

	view := model.View()
	assert.Contains(t, view, "default") // Placeholder should appear
	assert.Contains(t, view, "Enter:")
}
