package notes

import (
	"testing"

	"github.com/Paintersrp/an/internal/search"
)

func TestListItemFilterValueUsesTitle(t *testing.T) {
	t.Parallel()

	item := ListItem{
		title:        "Front Matter Title",
		tags:         []string{"project", "draft"},
		subdirectory: "notes",
	}

	got := item.FilterValue()
	want := "Front Matter Title [project draft] [notes]"

	if got != want {
		t.Fatalf("FilterValue() = %q, want %q", got, want)
	}
}

func TestListItemFilterValueFallsBackToFilename(t *testing.T) {
	t.Parallel()

	item := ListItem{
		fileName: "meeting-notes.md",
	}

	got := item.FilterValue()
	want := "meeting-notes [] []"

	if got != want {
		t.Fatalf("FilterValue() = %q, want %q", got, want)
	}
}

func TestListItemFilterValueIncludesHighlight(t *testing.T) {
	t.Parallel()

	store := newHighlightStore()
	path := "/tmp/note.md"
	store.setAll(map[string]search.Result{
		path: search.Result{Snippet: "matched snippet"},
	})

	item := ListItem{
		fileName:   "note.md",
		path:       path,
		highlights: store,
	}

	got := item.FilterValue()
	want := "note [] [] matched snippet"

	if got != want {
		t.Fatalf("FilterValue() = %q, want %q", got, want)
	}
}

func TestListItemDescriptionIncludesHighlight(t *testing.T) {
	t.Parallel()

	store := newHighlightStore()
	path := "/tmp/note.md"
	store.setAll(map[string]search.Result{
		path: search.Result{Snippet: "body match"},
	})

	item := ListItem{
		fileName:   "note.md",
		highlights: store,
		path:       path,
	}

	got := item.Description()
	want := "No tags\nbody match"

	if got != want {
		t.Fatalf("Description() = %q, want %q", got, want)
	}
}
