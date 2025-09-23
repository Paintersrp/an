package note

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Paintersrp/an/internal/templater"
)

func TestCreateCleansUpOnTemplateError(t *testing.T) {
	t.Parallel()

	vaultDir := t.TempDir()
	subDir := filepath.Join("foo", "bar")
	note := NewZettelkastenNote(vaultDir, subDir, "test-note", nil, nil, "")

	tmpl, err := templater.NewTemplater()
	if err != nil {
		t.Fatalf("failed to create templater: %v", err)
	}

	created, err := note.Create("nonexistent-template", tmpl, "")
	if err == nil {
		t.Fatalf("expected template execution error, got nil")
	}

	if created {
		t.Fatalf("expected note creation to fail")
	}

	if _, err := os.Stat(note.GetFilepath()); !os.IsNotExist(err) {
		t.Fatalf("expected note file to be removed, got err %v", err)
	}

	deepestDir := filepath.Join(vaultDir, subDir)
	if _, err := os.Stat(deepestDir); !os.IsNotExist(err) {
		t.Fatalf("expected deepest directory to be removed, got err %v", err)
	}

	parentDir := filepath.Join(vaultDir, "foo")
	if _, err := os.Stat(parentDir); !os.IsNotExist(err) {
		t.Fatalf("expected parent directory to be removed, got err %v", err)
	}
}
