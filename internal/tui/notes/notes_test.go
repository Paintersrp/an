package notes

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"

	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/views"
)

func TestCycleViewOrder(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	fileHandler := handler.NewFileHandler(tempDir)
	viewManager := views.NewViewManager(fileHandler, tempDir)

	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)

	model := &NoteListModel{
		list:      l,
		state:     &state.State{Handler: fileHandler, ViewManager: viewManager, Vault: tempDir},
		viewName:  "default",
		sortField: sortByTitle,
		sortOrder: ascending,
	}

	expectedOrder := []string{"unfulfilled", "archive", "orphan", "trash", "default"}

	for i, want := range expectedOrder {
		model.cycleView()
		if got := model.viewName; got != want {
			t.Fatalf("step %d: expected view %q, got %q", i+1, want, got)
		}
	}
}
