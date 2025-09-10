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
	// Should show first 10 items with scrolling behavior
	assert.Contains(t, view, "Item 0")
	assert.Contains(t, view, "Item 9")
	// Should not show item 10 (beyond viewport)
	assert.NotContains(t, view, "Item 10")
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

func TestFilterableSelectModel_Scrolling(t *testing.T) {
	// Create model with 15 items (more than the 10-item viewport)
	items := []string{}
	for i := 0; i < 15; i++ {
		items = append(items, fmt.Sprintf("Item %d", i))
	}
	model := newFilterableSelectModel("Choose:", items)

	// Test initial state - viewport should be at 0
	assert.Equal(t, 0, model.viewport)
	assert.Equal(t, 0, model.selected)

	// Move down to item 5 - should still be in viewport
	for i := 0; i < 5; i++ {
		result, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = result.(FilterableSelectModel)
	}
	assert.Equal(t, 5, model.selected)
	assert.Equal(t, 0, model.viewport) // Viewport hasn't moved yet

	// Move down to item 10 - should trigger viewport scroll
	for i := 0; i < 5; i++ {
		result, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = result.(FilterableSelectModel)
	}
	assert.Equal(t, 10, model.selected)
	assert.Equal(t, 1, model.viewport) // Viewport scrolled down

	// Move up - should scroll viewport back
	for i := 0; i < 5; i++ {
		result, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
		model = result.(FilterableSelectModel)
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

func TestFilterableSelectModel_ScrollingWithFilter(t *testing.T) {
	// Create model with many items that will be filtered
	items := []string{}
	for i := 0; i < 15; i++ {
		items = append(items, fmt.Sprintf("Test Item %d", i))
	}
	model := newFilterableSelectModel("Choose:", items)

	// Apply filter - should reset viewport to 0
	model.filterInput.SetValue("Test")
	model.filterOptions()
	assert.Equal(t, 0, model.viewport)
	assert.Equal(t, 0, model.selected)

	// All items match filter, should have 15 filtered items
	assert.Equal(t, 15, len(model.filtered))

	// Navigate down beyond viewport
	for i := 0; i < 12; i++ {
		result, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = result.(FilterableSelectModel)
	}
	assert.Equal(t, 12, model.selected)
	assert.Equal(t, 3, model.viewport) // Should have scrolled
}