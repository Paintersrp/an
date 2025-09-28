package search

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func writeNote(t testing.TB, dir, name, content string) string {
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

func TestIndexUpdateHandlesChangesAndRemovals(t *testing.T) {
	dir := t.TempDir()
	path := writeNote(t, dir, "note.md", "---\ntitle: First\n---\noriginal content")

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build([]string{path}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	// Modify the note and update using a relative path to ensure normalization
	// succeeds.
	updated := "---\ntitle: First\n---\noriginal content with updated term"
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("rewrite note: %v", err)
	}

	rel, err := filepath.Rel(dir, path)
	if err != nil {
		t.Fatalf("filepath.Rel returned error: %v", err)
	}

	if err := idx.Update(rel); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	results := idx.Search(Query{Term: "updated"})
	if len(results) != 1 {
		t.Fatalf("expected updated note to be searchable, got %+v", results)
	}
	if results[0].Path != filepath.Clean(path) {
		t.Fatalf("expected result path %s, got %s", filepath.Clean(path), results[0].Path)
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove note: %v", err)
	}

	if err := idx.Update(path); err != nil {
		t.Fatalf("Update after removal returned error: %v", err)
	}

	results = idx.Search(Query{Term: "updated"})
	if len(results) != 0 {
		t.Fatalf("expected removed note to disappear from index, got %+v", results)
	}
}

func TestIndexCloneProducesIndependentCopy(t *testing.T) {
	dir := t.TempDir()
	path := writeNote(t, dir, "note.md", "---\ntitle: First\n---\noriginal content")

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build([]string{path}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	clone := idx.Clone()
	if clone == nil {
		t.Fatalf("Clone returned nil")
	}

	if got, want := len(clone.Documents()), len(idx.Documents()); got != want {
		t.Fatalf("expected clone to copy documents, got %d want %d", got, want)
	}

	// Mutate the original index and ensure the clone is unaffected.
	if err := idx.Remove(path); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if got := len(idx.Documents()); got != 0 {
		t.Fatalf("expected original index to drop document, got %d", got)
	}

	if got := len(clone.Documents()); got != 1 {
		t.Fatalf("expected clone to preserve document, got %d", got)
	}

	// Mutate the clone and ensure the original remains unchanged.
	if err := clone.Remove(path); err != nil {
		t.Fatalf("Remove on clone returned error: %v", err)
	}

	if got := len(clone.Documents()); got != 0 {
		t.Fatalf("expected clone removal to drop document, got %d", got)
	}

	if got := len(idx.Documents()); got != 0 {
		t.Fatalf("expected original index to remain empty, got %d", got)
	}
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
	unrelated := writeNote(t, dir, "projects/reference.md", "---\ntitle: Reference\ntags:\n  - reference\nstatus: planned\n---\nReference content\n")

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build([]string{matched, archived, unrelated}); err != nil {
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

	// Tag filters should require only one matching value.
	tagOnly := Query{Tags: []string{"project", "reference"}}
	results = idx.Search(tagOnly)
	wantTags := map[string]struct{}{
		filepath.Clean(matched):   {},
		filepath.Clean(archived):  {},
		filepath.Clean(unrelated): {},
	}
	if len(results) != len(wantTags) {
		t.Fatalf("expected %d tag matches, got %+v", len(wantTags), results)
	}
	for _, res := range results {
		if _, ok := wantTags[res.Path]; !ok {
			t.Fatalf("unexpected tag result %+v", res)
		}
		delete(wantTags, res.Path)
	}
	if len(wantTags) != 0 {
		t.Fatalf("tag filter missing expected notes: %+v", wantTags)
	}
}

func TestIndexSearchFiltersMatchAnySelectedValues(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	projectActive := writeNote(t, dir, "notes/alpha.md", "---\ntitle: Alpha\ntags:\n  - project\n  - urgent\nstatus: active\n---\nAlpha body\n")
	planningStalled := writeNote(t, dir, "notes/beta.md", "---\ntitle: Beta\ntags:\n  - planning\nstatus: stalled\n---\nBeta body\n")
	referenceDone := writeNote(t, dir, "notes/gamma.md", "---\ntitle: Gamma\ntags:\n  - reference\nstatus: done\n---\nGamma body\n")
	unrelatedTag := writeNote(t, dir, "notes/delta.md", "---\ntitle: Delta\ntags:\n  - urgent\nstatus: active\n---\nDelta body\n")

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build([]string{projectActive, planningStalled, referenceDone, unrelatedTag}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	query := Query{
		Tags: []string{"project", "planning"},
		Metadata: map[string][]string{
			"status": []string{"active", "stalled"},
		},
	}

	results := idx.Search(query)
	want := map[string]struct{}{
		filepath.Clean(projectActive):   {},
		filepath.Clean(planningStalled): {},
	}
	if len(results) != len(want) {
		t.Fatalf("expected %d matching notes, got %+v", len(want), results)
	}
	for _, res := range results {
		if _, ok := want[res.Path]; !ok {
			t.Fatalf("unexpected result %+v", res)
		}
		delete(want, res.Path)
	}
	if len(want) != 0 {
		t.Fatalf("missing expected results: %+v", want)
	}

	// Metadata-only queries should return all notes with any selected value.
	metadataOnly := Query{Metadata: map[string][]string{"status": []string{"active", "stalled"}}}
	results = idx.Search(metadataOnly)
	want = map[string]struct{}{
		filepath.Clean(projectActive):   {},
		filepath.Clean(planningStalled): {},
		filepath.Clean(unrelatedTag):    {},
	}
	if len(results) != len(want) {
		t.Fatalf("expected %d metadata matches, got %+v", len(want), results)
	}
	for _, res := range results {
		if _, ok := want[res.Path]; !ok {
			t.Fatalf("unexpected metadata result %+v", res)
		}
		delete(want, res.Path)
	}
	if len(want) != 0 {
		t.Fatalf("metadata results missing expected notes: %+v", want)
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

func TestIndexRelatedResolvesRelativeLinks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	parent := writeNote(t, dir, "notes/parent.md", "Parent note\n")
	child := writeNote(t, dir, "notes/nested/child.md", "Links to [parent](../parent.md) and [sibling](./sibling.md)\n")
	sibling := writeNote(t, dir, "notes/nested/sibling.md", "Back to [child](./child.md)\n")

	idx := NewIndex(dir, Config{})
	if err := idx.Build([]string{parent, child, sibling}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	relatedChild := idx.Related(child)
	wantChildOutbound := []string{filepath.Clean(parent), filepath.Clean(sibling)}
	sort.Strings(wantChildOutbound)
	if diff := cmpSlices(relatedChild.Outbound, wantChildOutbound); diff != "" {
		t.Fatalf("unexpected outbound for child: %s", diff)
	}
	wantChildBacklinks := []string{filepath.Clean(sibling)}
	if diff := cmpSlices(relatedChild.Backlinks, wantChildBacklinks); diff != "" {
		t.Fatalf("unexpected backlinks for child: %s", diff)
	}

	relatedParent := idx.Related(parent)
	wantParentBacklinks := []string{filepath.Clean(child)}
	if diff := cmpSlices(relatedParent.Backlinks, wantParentBacklinks); diff != "" {
		t.Fatalf("unexpected backlinks for parent: %s", diff)
	}

	relatedSibling := idx.Related(sibling)
	wantSiblingOutbound := []string{filepath.Clean(child)}
	if diff := cmpSlices(relatedSibling.Outbound, wantSiblingOutbound); diff != "" {
		t.Fatalf("unexpected outbound for sibling: %s", diff)
	}
	wantSiblingBacklinks := []string{filepath.Clean(child)}
	if diff := cmpSlices(relatedSibling.Backlinks, wantSiblingBacklinks); diff != "" {
		t.Fatalf("unexpected backlinks for sibling: %s", diff)
	}
}

func TestIndexSearchRanksFrequencyAndTags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	common := time.Now().Add(-time.Hour)
	highFreq := writeNote(t, dir, "projects/high.md", "---\ntags:\n  - project\n---\nalpha alpha alpha focus\n")
	lowFreq := writeNote(t, dir, "projects/low.md", "---\ntags:\n  - project\n---\nalpha details\n")

	if err := os.Chtimes(highFreq, common, common); err != nil {
		t.Fatalf("chtimes high freq: %v", err)
	}
	if err := os.Chtimes(lowFreq, common, common); err != nil {
		t.Fatalf("chtimes low freq: %v", err)
	}

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build([]string{highFreq, lowFreq}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	query := Query{Term: "alpha", Tags: []string{"project"}}
	results := idx.Search(query)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Path != filepath.Clean(highFreq) {
		t.Fatalf("expected high frequency note first, got %s", results[0].Path)
	}
	if results[0].Score <= results[1].Score {
		t.Fatalf("expected higher score for frequent note, got %+v", results)
	}
}

func TestIndexSearchPrioritizesRecency(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	recent := writeNote(t, dir, "notes/recent.md", "---\ntitle: Recent\n---\nterm here\n")
	older := writeNote(t, dir, "notes/older.md", "---\ntitle: Older\n---\nterm here\n")

	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(older, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes older: %v", err)
	}

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build([]string{recent, older}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	results := idx.Search(Query{Term: "term"})
	if len(results) != 2 {
		t.Fatalf("expected two results, got %d", len(results))
	}

	if results[0].Path != filepath.Clean(recent) {
		t.Fatalf("expected recent note first, got %s", results[0].Path)
	}
	if results[0].Score <= results[1].Score {
		t.Fatalf("expected recent note to score higher, got %+v", results)
	}
}

func TestIndexSearchFuzzyMatchesTitlesAndHeadings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	note := writeNote(t, dir, "plans/launch.md", "---\ntitle: Launch Plan\n---\n# Launch Checklist\nDetails here\n")

	idx := NewIndex(dir, Config{EnableBody: false})
	if err := idx.Build([]string{note}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	results := idx.Search(Query{Term: "lanch plan"})
	if len(results) != 1 {
		t.Fatalf("expected fuzzy match result, got %d", len(results))
	}
	if results[0].MatchFrom != "fuzzy" {
		t.Fatalf("expected fuzzy match, got %q", results[0].MatchFrom)
	}
	if !strings.Contains(results[0].Snippet, "Launch") {
		t.Fatalf("expected snippet to reference heading or title, got %q", results[0].Snippet)
	}
	if results[0].Score <= 0 {
		t.Fatalf("expected fuzzy match to have positive score, got %+v", results[0])
	}
}

func TestIndexSearchResultsIncludeRelatedNotes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := writeNote(t, dir, "notes/source.md", "[[target]]\nFocus term\n")
	target := writeNote(t, dir, "notes/target.md", "---\ntitle: Target\n---\nterm reference\n")

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build([]string{source, target}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	results := idx.Search(Query{Term: "term"})
	if len(results) != 2 {
		t.Fatalf("expected two matches, got %d", len(results))
	}

	for _, res := range results {
		if len(res.Related.Outbound) == 0 && len(res.Related.Backlinks) == 0 {
			t.Fatalf("expected related notes for %s, got %+v", res.Path, res.Related)
		}
	}
}

func BenchmarkIndexSearchRanked(b *testing.B) {
	dir := b.TempDir()
	var paths []string
	for i := 0; i < 200; i++ {
		content := fmt.Sprintf("---\ntitle: Note %d\ntags:\n  - project\n---\nbody project term %d\n", i, i)
		path := writeNote(b, dir, fmt.Sprintf("note-%03d.md", i), content)
		paths = append(paths, path)
	}

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build(paths); err != nil {
		b.Fatalf("Build returned error: %v", err)
	}

	query := Query{Term: "project"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = idx.Search(query)
	}
}

func BenchmarkIndexSearchAlphabetical(b *testing.B) {
	dir := b.TempDir()
	var paths []string
	for i := 0; i < 200; i++ {
		content := fmt.Sprintf("---\ntitle: Baseline %d\n---\nbody baseline term %d\n", i, i)
		path := writeNote(b, dir, fmt.Sprintf("baseline-%03d.md", i), content)
		paths = append(paths, path)
	}

	idx := NewIndex(dir, Config{EnableBody: true})
	if err := idx.Build(paths); err != nil {
		b.Fatalf("Build returned error: %v", err)
	}

	query := Query{Term: "baseline"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = alphabeticalSearch(idx, query)
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

func alphabeticalSearch(idx *Index, q Query) []Result {
	term := strings.TrimSpace(q.Term)
	loweredTerm := strings.ToLower(term)

	results := make([]Result, 0)
	for _, doc := range idx.docs {
		if !doc.matchesFilters(q) {
			continue
		}

		if loweredTerm == "" {
			results = append(results, Result{Path: doc.Path, MatchFrom: "metadata"})
			continue
		}

		if snippet, freq := doc.matchFrontMatter(loweredTerm); freq > 0 {
			results = append(results, Result{Path: doc.Path, Snippet: snippet, MatchFrom: "frontmatter"})
			continue
		}

		if snippet, freq := doc.matchLinks(loweredTerm); freq > 0 {
			results = append(results, Result{Path: doc.Path, Snippet: snippet, MatchFrom: "links"})
			continue
		}

		if idx.cfg.EnableBody {
			if snippet, freq := doc.matchBody(loweredTerm); freq > 0 {
				results = append(results, Result{Path: doc.Path, Snippet: snippet, MatchFrom: "body"})
				continue
			}
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})
	return results
}
