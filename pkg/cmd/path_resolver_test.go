package cmd

import (
	"path/filepath"
	"testing"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
	"github.com/spf13/cobra"
)

func TestResolveVaultPath(t *testing.T) {
	vaultDir := t.TempDir()

	cfg := &config.Config{
		Workspaces: map[string]*config.Workspace{
			"default": {VaultDir: vaultDir},
		},
		CurrentWorkspace: "default",
	}
	if err := cfg.ActivateWorkspace("default"); err != nil {
		t.Fatalf("failed to activate workspace: %v", err)
	}
	st := &state.State{Config: cfg, Workspace: cfg.MustWorkspace(), WorkspaceName: cfg.CurrentWorkspace}

	tests := map[string]struct {
		command *cobra.Command
		input   string
		want    string
		wantErr bool
	}{
		"absolute inside vault": {
			command: &cobra.Command{Use: "archive"},
			input:   filepath.Join(vaultDir, "note.md"),
			want:    filepath.Join(vaultDir, "note.md"),
		},
		"relative inside vault": {
			command: &cobra.Command{Use: "archive"},
			input:   "note.md",
			want:    filepath.Join(vaultDir, "note.md"),
		},
		"escape attempt": {
			command: &cobra.Command{Use: "archive"},
			input:   "../evil.md",
			wantErr: true,
		},
		"unarchive infers archive directory": {
			command: &cobra.Command{Use: "unarchive"},
			input:   "restored.md",
			want:    filepath.Join(vaultDir, "archive", "restored.md"),
		},
		"unarchive respects explicit archive prefix": {
			command: &cobra.Command{Use: "unarchive"},
			input:   filepath.Join("archive", "restored.md"),
			want:    filepath.Join(vaultDir, "archive", "restored.md"),
		},
		"untrash infers trash directory": {
			command: &cobra.Command{Use: "untrash"},
			input:   "restored.md",
			want:    filepath.Join(vaultDir, "trash", "restored.md"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ResolveVaultPath(tc.command, st, tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveVaultPath returned error: %v", err)
			}
			if got != filepath.Clean(tc.want) {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
