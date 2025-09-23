package views

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/handler"
)

func TestGetFilesByView_DefaultAndArchive(t *testing.T) {
	vaultDir := t.TempDir()

	mustMkdirAll(t, filepath.Join(vaultDir, "archive", "project"))
	mustMkdirAll(t, filepath.Join(vaultDir, "trash", "project"))
	mustMkdirAll(t, filepath.Join(vaultDir, "notes"))

	keepPath := filepath.Join(vaultDir, "notes", "keep.md")
	skipPath := filepath.Join(vaultDir, "notes", "skip.md")
	archivedPath := filepath.Join(vaultDir, "archive", "project", "archived.md")
	trashedPath := filepath.Join(vaultDir, "trash", "project", "trashed.md")

	mustWriteFile(t, keepPath)
	mustWriteFile(t, skipPath)
	mustWriteFile(t, archivedPath)
	mustWriteFile(t, trashedPath)

	h := handler.NewFileHandler(vaultDir)
	ws := &config.Workspace{
		VaultDir: vaultDir,
		Views: map[string]config.ViewDefinition{
			"custom": {
				Exclude: []string{"archive", "trash", "notes/skip.md"},
			},
		},
	}
	cfg := &config.Config{
		Workspaces:       map[string]*config.Workspace{"default": ws},
		CurrentWorkspace: "default",
	}
	if err := cfg.ActivateWorkspace("default"); err != nil {
		t.Fatalf("failed to activate workspace: %v", err)
	}

	vm, err := NewViewManager(h, cfg)
	if err != nil {
		t.Fatalf("NewViewManager returned error: %v", err)
	}

	t.Run("default view excludes archive and trash", func(t *testing.T) {
		files, err := vm.GetFilesByView("default")
		if err != nil {
			t.Fatalf("GetFilesByView returned error: %v", err)
		}

		if !contains(files, keepPath) {
			t.Fatalf("default view missing expected file %s", keepPath)
		}

		if contains(files, archivedPath) {
			t.Fatalf("default view unexpectedly contained archived file %s", archivedPath)
		}

		if contains(files, trashedPath) {
			t.Fatalf("default view unexpectedly contained trashed file %s", trashedPath)
		}
	})

	t.Run("archive view returns archived notes", func(t *testing.T) {
		files, err := vm.GetFilesByView("archive")
		if err != nil {
			t.Fatalf("GetFilesByView returned error: %v", err)
		}

		if !contains(files, archivedPath) {
			t.Fatalf("archive view missing expected file %s", archivedPath)
		}

		if contains(files, keepPath) {
			t.Fatalf("archive view unexpectedly contained active file %s", keepPath)
		}
	})

	t.Run("custom view excludes configured files", func(t *testing.T) {
		files, err := vm.GetFilesByView("custom")
		if err != nil {
			t.Fatalf("GetFilesByView returned error: %v", err)
		}

		if contains(files, skipPath) {
			t.Fatalf("custom view unexpectedly contained excluded file %s", skipPath)
		}

		if contains(files, trashedPath) {
			t.Fatalf("custom view unexpectedly contained trashed file %s", trashedPath)
		}

		if contains(files, archivedPath) {
			t.Fatalf("custom view unexpectedly contained archived file %s", archivedPath)
		}

		if !contains(files, keepPath) {
			t.Fatalf("custom view missing expected file %s", keepPath)
		}
	})

	t.Run("invalid view returns error", func(t *testing.T) {
		if _, err := vm.GetFilesByView("unknown"); err == nil {
			t.Fatal("expected error for unknown view, got nil")
		}
	})
}

func TestViewManagerOrderHonorsConfig(t *testing.T) {
	t.Parallel()

	vaultDir := t.TempDir()
	h := handler.NewFileHandler(vaultDir)
	ws := &config.Workspace{
		VaultDir: vaultDir,
		Views: map[string]config.ViewDefinition{
			"custom": {},
			"beta":   {},
		},
		ViewOrder: []string{"custom", "beta"},
	}
	cfg := &config.Config{
		Workspaces:       map[string]*config.Workspace{"default": ws},
		CurrentWorkspace: "default",
	}
	if err := cfg.ActivateWorkspace("default"); err != nil {
		t.Fatalf("failed to activate workspace: %v", err)
	}

	vm, err := NewViewManager(h, cfg)
	if err != nil {
		t.Fatalf("NewViewManager returned error: %v", err)
	}

	order := vm.Order()
	if len(order) == 0 {
		t.Fatalf("expected order to contain views, got %v", order)
	}

	if order[0] != "custom" {
		t.Fatalf("expected first view to be 'custom', got %q", order[0])
	}

	if order[1] != "beta" {
		t.Fatalf("expected second view to be 'beta', got %q", order[1])
	}

	seen := make(map[string]struct{}, len(order))
	for _, name := range order {
		if _, ok := seen[name]; ok {
			t.Fatalf("found duplicate view %q in order %v", name, order)
		}
		seen[name] = struct{}{}
	}
}

func TestGetTitleForView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		viewFlag  string
		sortField SortField
		sortOrder SortOrder
		want      string
	}{{
		name:      "known view and sort field ascending",
		viewFlag:  "default",
		sortField: SortFieldTitle,
		sortOrder: SortOrderAscending,
		want:      "✅ - All View \nSort: Title (Ascending)",
	}, {
		name:      "unknown view falls back to default prefix",
		viewFlag:  "mystery",
		sortField: SortField("mystery"),
		sortOrder: SortOrderDescending,
		want:      "✅ - All View \nSort: Unknown (Descending)",
	}}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := GetTitleForView(tc.viewFlag, tc.sortField, tc.sortOrder)
			if got != tc.want {
				t.Fatalf("GetTitleForView(%q, %q, %q) = %q, want %q",
					tc.viewFlag, tc.sortField, tc.sortOrder, got, tc.want)
			}
		})
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
