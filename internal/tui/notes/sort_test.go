package notes

import "testing"

func TestSortItemsTitleFallsBackToFilename(t *testing.T) {
	items := []ListItem{
		{fileName: "beta.md"},
		{fileName: "alpha.md"},
		{fileName: "gamma.md", title: "gamma note"},
	}

	sorted := sortItems(items, sortByTitle, ascending)

	got := make([]ListItem, len(sorted))
	for i, item := range sorted {
		listItem, ok := item.(ListItem)
		if !ok {
			t.Fatalf("expected ListItem, got %T", item)
		}
		got[i] = listItem
	}

	if got[0].fileName != "alpha.md" {
		t.Fatalf("expected first item to be alpha.md, got %q", got[0].fileName)
	}

	if got[1].fileName != "beta.md" {
		t.Fatalf("expected second item to be beta.md, got %q", got[1].fileName)
	}

	if got[2].title != "gamma note" {
		t.Fatalf("expected third item to keep title 'gamma note', got %q", got[2].title)
	}
}

func TestSortItemsTitleFallsBackToFilenameDescending(t *testing.T) {
	items := []ListItem{
		{fileName: "beta.md"},
		{fileName: "alpha.md"},
		{fileName: "gamma.md", title: "gamma note"},
	}

	sorted := sortItems(items, sortByTitle, descending)

	got := make([]ListItem, len(sorted))
	for i, item := range sorted {
		listItem, ok := item.(ListItem)
		if !ok {
			t.Fatalf("expected ListItem, got %T", item)
		}
		got[i] = listItem
	}

	if got[0].title != "gamma note" {
		t.Fatalf("expected first item to have title 'gamma note', got %q", got[0].title)
	}

	if got[1].fileName != "beta.md" {
		t.Fatalf("expected second item to be beta.md, got %q", got[1].fileName)
	}

	if got[2].fileName != "alpha.md" {
		t.Fatalf("expected third item to be alpha.md, got %q", got[2].fileName)
	}
}
