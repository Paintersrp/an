package notes

import "testing"

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
