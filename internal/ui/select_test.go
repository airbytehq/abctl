package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestSelectModel_Init(t *testing.T) {
	model := newSelectModel("Choose:", []string{"opt1", "opt2"})

	cmd := model.Init()
	assert.Nil(t, cmd)
}

func TestSelectModel_View(t *testing.T) {
	model := newSelectModel("Choose an option:", []string{"option1", "option2", "option3"})

	view := model.View()

	// Should contain prompt
	if !strings.Contains(view, "Choose an option:") {
		t.Error("View should contain prompt")
	}

	// Should contain options
	if !strings.Contains(view, "option1") {
		t.Error("View should contain option1")
	}

	// Should show selection cursor on first option
	if !strings.Contains(view, "> option1") {
		t.Error("View should show cursor on first option")
	}

	// Should contain instructions
	if !strings.Contains(view, "Press enter to select") {
		t.Error("View should contain instructions")
	}
}

func TestSelectModel_Navigation(t *testing.T) {
	model := newSelectModel("Choose:", []string{"opt1", "opt2", "opt3"})

	// Test down navigation
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = result.(SelectModel)
	if model.selected != 1 {
		t.Errorf("Expected selected=1 after down, got %d", model.selected)
	}

	// Test up navigation
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = result.(SelectModel)
	if model.selected != 0 {
		t.Errorf("Expected selected=0 after up, got %d", model.selected)
	}

	// Test down arrow
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = result.(SelectModel)
	if model.selected != 1 {
		t.Errorf("Expected selected=1 after down arrow, got %d", model.selected)
	}
}

func TestSelectModel_BoundaryConditions(t *testing.T) {
	model := newSelectModel("Choose:", []string{"opt1", "opt2"})

	// Test up at top boundary - should stay at 0
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = result.(SelectModel)
	if model.selected != 0 {
		t.Errorf("Expected selected=0 when up at boundary, got %d", model.selected)
	}

	// Go to bottom
	model.selected = 1
	// Test down at bottom boundary - should stay at 1
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = result.(SelectModel)
	if model.selected != 1 {
		t.Errorf("Expected selected=1 when down at boundary, got %d", model.selected)
	}
}

func TestSelectModel_Selection(t *testing.T) {
	model := newSelectModel("Choose:", []string{"opt1", "opt2"})

	// Test enter key
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(SelectModel)

	assert.NotNil(t, cmd)
	assert.False(t, model.cancelled)
}

func TestSelectModel_Cancel(t *testing.T) {
	model := newSelectModel("Choose:", []string{"opt1", "opt2"})

	// Test escape key
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = result.(SelectModel)

	assert.NotNil(t, cmd)
	assert.True(t, model.cancelled)
}

func TestSelectModel_Scrolling(t *testing.T) {
	// Create model with 15 items (more than the 10-item viewport)
	items := []string{}
	for i := 0; i < 15; i++ {
		items = append(items, fmt.Sprintf("Item %d", i))
	}
	model := newSelectModel("Choose:", items)

	// Test initial state - viewport should be at 0
	assert.Equal(t, 0, model.viewport)
	assert.Equal(t, 0, model.selected)

	// Move down to item 5 - should still be in viewport
	for i := 0; i < 5; i++ {
		result, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = result.(SelectModel)
	}
	assert.Equal(t, 5, model.selected)
	assert.Equal(t, 0, model.viewport) // Viewport hasn't moved yet

	// Move down to item 10 - should trigger viewport scroll
	for i := 0; i < 5; i++ {
		result, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = result.(SelectModel)
	}
	assert.Equal(t, 10, model.selected)
	assert.Equal(t, 1, model.viewport) // Viewport scrolled down

	// Move up - should scroll viewport back
	for i := 0; i < 5; i++ {
		result, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
		model = result.(SelectModel)
	}
	assert.Equal(t, 5, model.selected)
	// Viewport should still be 1 since selected is 5 and viewport shows items 1-10
	assert.Equal(t, 1, model.viewport) 

	// Test view shows correct items for current viewport (1-10)
	view := model.View()
	assert.Contains(t, view, "Item 1") // Should show first item in viewport
	assert.Contains(t, view, "Item 10") // Should show last item in viewport
	assert.NotContains(t, view, "Item 0") // Should not show items outside viewport
	assert.NotContains(t, view, "Item 11") // Should not show items outside viewport
}
