package submodels

import (
	"testing"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
)

func TestInputModelUpdateHandlesKeyMessages(t *testing.T) {
	m := NewInputModel()

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated, ok := model.(*InputModel)
	if !ok {
		t.Fatalf("expected *InputModel, got %T", model)
	}

	if updated.cursorMode != cursor.CursorStatic {
		t.Fatalf("expected cursor mode %v, got %v", cursor.CursorStatic, updated.cursorMode)
	}

	nextModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	next, ok := nextModel.(*InputModel)
	if !ok {
		t.Fatalf("expected *InputModel, got %T", nextModel)
	}

	if next.Input.Value() != "a" {
		t.Fatalf("expected input value %q, got %q", "a", next.Input.Value())
	}
}
