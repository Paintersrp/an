package search

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func writeNote(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
	return path
}

func TestIndexSearchBodyRespectsToggle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	note := writeNote(t, dir, "note.md", "---\ntitle: Example\n---\nbody term here")

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build([]string{note}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	results := idx.Search(Query{Term: "term"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result with body search enabled, got %d", len(results))
	}

	idx = NewIndex(dir, Config{EnableBody: false})
	if err := idx.Build([]string{note}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	results = idx.Search(Query{Term: "term"})
	if len(results) != 0 {
		t.Fatalf("expected 0 results with body search disabled, got %d", len(results))
	}
}

func TestIndexSearchIgnoredFolders(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	included := writeNote(t, dir, "keep/note.md", "---\ntitle: Keep\n---\nbody")
	ignored := writeNote(t, dir, "archive/skip.md", "---\ntitle: Skip\n---\nbody skip")

	idx := NewIndex(dir, Config{EnableBody: true, IgnoredFolders: []string{"archive"}})
	if err := idx.Build([]string{included, ignored}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	results := idx.Search(Query{Term: "skip"})
	if len(results) != 0 {
		t.Fatalf("expected ignored folder to be skipped, got %d results", len(results))
	}

	results = idx.Search(Query{Term: "keep"})
	if len(results) != 1 {
		t.Fatalf("expected included note to be searchable, got %d results", len(results))
	}
}

func TestIndexSearchSupportsMetadataAndTags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	matched := writeNote(t, dir, "projects/plan.md", "---\ntitle: Plan\ntags:\n  - project\n  - urgent\nstatus: active\n---\nMilestone body\n")
	archived := writeNote(t, dir, "projects/archive.md", "---\ntitle: Old\ntags:\n  - project\nstatus: done\n---\nFinished body\n")

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build([]string{matched, archived}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	query := Query{
		Term: "milestone",
		Tags: []string{"project"},
		Metadata: map[string][]string{
			"status": []string{"active"},
		},
	}

	results := idx.Search(query)
	if len(results) != 1 || results[0].Path != filepath.Clean(matched) {
		t.Fatalf("expected matching note, got %+v", results)
	}

	// Metadata-only queries should still filter results.
	query.Term = ""
	results = idx.Search(query)
	if len(results) != 1 || results[0].Path != filepath.Clean(matched) {
		t.Fatalf("expected metadata filters to return match, got %+v", results)
	}

	query.Metadata["status"] = []string{"missing"}
	results = idx.Search(query)
	if len(results) != 0 {
		t.Fatalf("expected metadata filters to exclude non-matching notes, got %+v", results)
	}
}

func TestIndexRelatedComputesBacklinksAndOutbound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	alpha := writeNote(t, dir, "alpha.md", "[[beta]]\n[[gamma]]\n")
	beta := writeNote(t, dir, "beta.md", "Content with [Alpha](alpha.md) and duplicate [[gamma]] reference\n")
	gamma := writeNote(t, dir, "notes/gamma.md", "No outbound links\n")
	orphan := writeNote(t, dir, "orphan.md", "External link [Example](https://example.com)\n")

	idx := NewIndex(dir, Config{})
	if err := idx.Build([]string{alpha, beta, gamma, orphan}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	relatedAlpha := idx.Related(alpha)
	wantAlphaOutbound := []string{filepath.Clean(beta), filepath.Clean(gamma)}
	if diff := cmpSlices(relatedAlpha.Outbound, wantAlphaOutbound); diff != "" {
		t.Fatalf("unexpected outbound for alpha: %s", diff)
	}
	wantAlphaBacklinks := []string{filepath.Clean(beta)}
	if diff := cmpSlices(relatedAlpha.Backlinks, wantAlphaBacklinks); diff != "" {
		t.Fatalf("unexpected backlinks for alpha: %s", diff)
	}

	relPath, err := filepath.Rel(dir, beta)
	if err != nil {
		t.Fatalf("filepath.Rel returned error: %v", err)
	}
	relatedBeta := idx.Related(relPath)
	wantBetaOutbound := []string{filepath.Clean(alpha), filepath.Clean(filepath.Join(dir, "notes", "gamma.md"))}
	if diff := cmpSlices(relatedBeta.Outbound, wantBetaOutbound); diff != "" {
		t.Fatalf("unexpected outbound for beta: %s", diff)
	}
	wantBetaBacklinks := []string{filepath.Clean(alpha)}
	if diff := cmpSlices(relatedBeta.Backlinks, wantBetaBacklinks); diff != "" {
		t.Fatalf("unexpected backlinks for beta: %s", diff)
	}

	relatedGamma := idx.Related(gamma)
	if len(relatedGamma.Outbound) != 0 {
		t.Fatalf("expected no outbound links for gamma, got %+v", relatedGamma.Outbound)
	}
	wantGammaBacklinks := []string{filepath.Clean(alpha), filepath.Clean(beta)}
	if diff := cmpSlices(relatedGamma.Backlinks, wantGammaBacklinks); diff != "" {
		t.Fatalf("unexpected backlinks for gamma: %s", diff)
	}

	relatedOrphan := idx.Related(orphan)
	if len(relatedOrphan.Outbound) != 0 || len(relatedOrphan.Backlinks) != 0 {
		t.Fatalf("expected orphan note to have no relationships, got %+v", relatedOrphan)
	}
}

func cmpSlices(got, want []string) string {
	if len(got) != len(want) {
		return fmt.Sprintf("length mismatch got %d want %d (got=%v, want=%v)", len(got), len(want), got, want)
	}
	for i := range got {
		if filepath.Clean(got[i]) != filepath.Clean(want[i]) {
			return fmt.Sprintf("value mismatch at %d: got %q want %q", i, got[i], want[i])
		}
	}
	return ""
}
