package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFilterableSelectModel_Init(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry"})

	cmd := model.Init()
	assert.NotNil(t, cmd) // Should focus text input
	assert.Equal(t, 0, model.selected)
	assert.Equal(t, []string{"Apple", "Banana", "Cherry"}, model.filtered)
}

func TestFilterableSelectModel_View(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana"})

	view := model.View()

	// Should contain prompt
	if !strings.Contains(view, "Choose:") {
		t.Error("View should contain prompt")
	}
	
	// Should contain filter instructions
	if !strings.Contains(view, "Start typing to filter") {
		t.Error("View should contain filter instructions")
	}
}

func TestFilterableSelectModel_Filtering(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Apricot", "Banana", "Cherry"})
	
	// Simulate typing "ap"
	model.filterInput.SetValue("ap")
	model.filterOptions()
	
	// Should filter to Apple and Apricot
	assert.Equal(t, []string{"Apple", "Apricot"}, model.filtered)
	assert.Equal(t, 0, model.selected) // Auto-select first match
}

func TestFilterableSelectModel_FilteringCaseInsensitive(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry"})
	
	// Simulate typing "APPLE" (uppercase)
	model.filterInput.SetValue("APPLE")
	model.filterOptions()
	
	// Should still find "Apple"
	assert.Equal(t, []string{"Apple"}, model.filtered)
	assert.Equal(t, 0, model.selected)
}

func TestFilterableSelectModel_NoMatches(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry"})
	
	// Simulate typing something that doesn't match
	model.filterInput.SetValue("xyz")
	model.filterOptions()
	
	// Should have no filtered options (nil slice)
	assert.Nil(t, model.filtered)
	assert.Equal(t, 0, model.selected)
}

func TestFilterableSelectModel_EmptyFilter(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry"})
	
	// Set some filter first
	model.filterInput.SetValue("ap")
	model.filterOptions()
	
	// Now clear the filter
	model.filterInput.SetValue("")
	model.filterOptions()
	
	// Should show all options again
	assert.Equal(t, []string{"Apple", "Banana", "Cherry"}, model.filtered)
}

func TestFilterableSelectModel_Navigation(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry"})

	// Test down arrow
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = result.(FilterableSelectModel)

	assert.Nil(t, cmd)
	assert.Equal(t, 1, model.selected)

	// Test up arrow
	result, cmd = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = result.(FilterableSelectModel)

	assert.Nil(t, cmd)
	assert.Equal(t, 0, model.selected)
}

func TestFilterableSelectModel_Selection(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry"})
	model.selected = 1

	// Test enter key
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(FilterableSelectModel)

	assert.NotNil(t, cmd)
	assert.Equal(t, 1, model.selected)
}

func TestFilterableSelectModel_Cancel(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry"})

	// Test escape key
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = result.(FilterableSelectModel)

	assert.NotNil(t, cmd)
	assert.True(t, model.cancelled)
}

func TestFilterableSelectModel_UpdateComprehensive(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry", "Date", "Elderberry"})
	
	// Test typing regular characters
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = result.(FilterableSelectModel)
	// The text input should handle this
	
	// Test Ctrl+C (should cancel)
	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model = result.(FilterableSelectModel)
	assert.NotNil(t, cmd)
	assert.True(t, model.cancelled)
	
	// Reset for next test
	model = newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry", "Date", "Elderberry"})
	
	// Test down arrow at bounds
	model.filtered = []string{"Apple", "Banana"}
	model.selected = 1
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = result.(FilterableSelectModel)
	assert.Equal(t, 1, model.selected) // Should stay at bottom (no wrap)
	
	// Test up arrow at bounds
	model.selected = 0
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = result.(FilterableSelectModel)
	assert.Equal(t, 0, model.selected) // Should stay at top (no wrap)
	
	// Test with empty filtered list
	model.filtered = []string{}
	model.selected = 0
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = result.(FilterableSelectModel)
	assert.Equal(t, 0, model.selected) // Should stay at 0
	
	// Test Tab key (should do nothing special, delegate to text input)
	model.filtered = []string{"Apple"}
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(FilterableSelectModel)
	// Just ensure no panic
}

func TestFilterableSelectModel_UpdateMoreCases(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry"})
	
	// Test Ctrl+D (should do nothing, pass to text input)
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	model = result.(FilterableSelectModel)
	// Just ensure no panic and model is returned
	assert.NotNil(t, model)
	
	// Test Page Down (should do nothing special)
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	model = result.(FilterableSelectModel)
	assert.NotNil(t, model)
	
	// Test unknown message type
	result, cmd := model.Update(struct{}{})
	model = result.(FilterableSelectModel)
	assert.Nil(t, cmd)
}

func TestNewFilterableSelectModel_EmptyOptions(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{})
	
	assert.Empty(t, model.options)
	assert.Empty(t, model.filtered)
	assert.Equal(t, "Start typing to filter...", model.filterInput.Placeholder)
}

func TestFilterableSelectModel_ViewWithMaxItems(t *testing.T) {
	// Create a model with many items
	items := []string{}
	for i := 0; i < 15; i++ {
		items = append(items, fmt.Sprintf("Item %d", i))
	}
	model := newFilterableSelectModel("Choose:", items)
	
	view := model.View()
	assert.Contains(t, view, "Choose:")
	// Should show "(5 more...)" text when there are many items
	assert.Contains(t, view, "(5 more...)")
}

func TestFilterableSelectModel_ViewStates(t *testing.T) {
	model := newFilterableSelectModel("Choose:", []string{"Apple", "Banana", "Cherry"})
	
	// Test with filtered results
	model.filterInput.SetValue("app")
	model.filterOptions()
	view := model.View()
	assert.Contains(t, view, "Choose:")
	assert.Contains(t, view, "Type to filter") // Check for actual text from View
	assert.Contains(t, view, "Apple") // Should show in results
	
	// Test with no results
	model.filterInput.SetValue("xyz")
	model.filterOptions()
	view = model.View()
	assert.Contains(t, view, "No matches found")
	
	// Test with selected item
	model.filterInput.SetValue("")
	model.filterOptions()
	model.selected = 1
	view = model.View()
	assert.Contains(t, view, "> Banana") // Selected item should have arrow
}