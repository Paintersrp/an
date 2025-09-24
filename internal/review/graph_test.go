package review

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Paintersrp/an/internal/search"
)

func TestBuildBacklinkGraph(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	alpha := writeNote(t, dir, "alpha.md", fixedTime())
	beta := writeNote(t, dir, "beta.md", fixedTime())
	gamma := writeNote(t, dir, "gamma.md", fixedTime())

	writeFile(t, alpha, "[[beta]]\n")
	writeFile(t, beta, "[[gamma]]\n[[alpha]]\n")
	writeFile(t, gamma, "No links\n")

	idx := search.NewIndex(dir, search.Config{})
	if err := idx.Build([]string{alpha, beta, gamma}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	graph := BuildBacklinkGraph(idx, []string{alpha})
	if len(graph.Nodes) == 0 {
		t.Fatalf("expected graph nodes, got 0")
	}

	node, ok := graph.Nodes[filepath.Clean(alpha)]
	if !ok {
		t.Fatalf("missing alpha node: %#v", graph.Nodes)
	}
	if len(node.Outbound) != 1 || node.Outbound[0] != filepath.Clean(beta) {
		t.Fatalf("unexpected outbound: %#v", node.Outbound)
	}
	if len(node.Backlinks) != 1 || node.Backlinks[0] != filepath.Clean(beta) {
		t.Fatalf("unexpected backlinks: %#v", node.Backlinks)
	}
}

func fixedTime() time.Time {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
