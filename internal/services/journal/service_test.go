package journal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/templater"
)

func newTemplater(t *testing.T, dir string) *templater.Templater {
	t.Helper()
	ws := &config.Workspace{VaultDir: dir}
	tmpl, err := templater.NewTemplater(ws)
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}
	return tmpl
}

func TestEnsureEntryCreatesNote(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(newTemplater(t, dir), handler.NewFileHandler(dir))

	entry, err := svc.EnsureEntry("day", 0, []string{"tag"}, nil, "body")
	if err != nil {
		t.Fatalf("EnsureEntry returned error: %v", err)
	}

	if entry.Path == "" {
		t.Fatalf("expected entry path to be populated")
	}
	if _, err := os.Stat(entry.Path); err != nil {
		t.Fatalf("expected journal entry file to exist: %v", err)
	}
}

func TestListReturnsSortedEntries(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(newTemplater(t, dir), handler.NewFileHandler(dir))

	now := time.Now()
	for i := 0; i < 3; i++ {
		date := now.AddDate(0, 0, -i).Format("20060102")
		name := filepath.Join(dir, "atoms", "day-"+date+".md")
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(name, []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	entries, err := svc.List("day")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected three entries, got %d", len(entries))
	}

	if entries[0].Date.Before(entries[1].Date) {
		t.Fatalf("expected entries to be sorted descending by date")
	}
}
