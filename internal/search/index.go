package search

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

type document struct {
	Path        string
	Tags        []string
	FrontMatter map[string][]string
	Links       []string
	Body        string
	ModifiedAt  time.Time
	Headings    []string
}

// Index stores searchable representations of notes on disk.
type Index struct {
	root string
	cfg  Config
	docs map[string]document
	// aliases maps lowercase note identifiers (relative paths, basenames,
	// and stemmed names) to their canonical on-disk path.
	aliases   map[string]string
	outbound  map[string][]string
	backlinks map[string][]string
}

// NewIndex constructs an empty index rooted at the provided directory.
func NewIndex(root string, cfg Config) *Index {
	return &Index{
		root:      filepath.Clean(root),
		cfg:       cfg,
		docs:      make(map[string]document),
		aliases:   make(map[string]string),
		outbound:  make(map[string][]string),
		backlinks: make(map[string][]string),
	}
}

// Build replaces the index contents using the provided note paths.
func (idx *Index) Build(paths []string) error {
	idx.docs = make(map[string]document, len(paths))
	idx.aliases = make(map[string]string)
	idx.outbound = make(map[string][]string)
	idx.backlinks = make(map[string][]string)
	for _, p := range paths {
		canonical := idx.normalize(p)
		if canonical == "" {
			continue
		}

		if idx.shouldIgnore(canonical) {
			continue
		}

		doc, err := idx.loadDocument(canonical)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("search: indexing %s: %w", canonical, err)
		}
		idx.docs[canonical] = doc
	}
	idx.refreshMetadata()
	return nil
}

// Update refreshes the indexed representation of the provided path.
//
// The method gracefully handles files that have been removed and ignores
// directories that fall under configured ignore rules.
func (idx *Index) Update(path string) error {
	if idx == nil {
		return nil
	}

	canonical := idx.normalize(path)
	if canonical == "" {
		return nil
	}

	if idx.shouldIgnore(canonical) {
		return idx.Remove(canonical)
	}

	doc, err := idx.loadDocument(canonical)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return idx.Remove(canonical)
		}
		return fmt.Errorf("search: indexing %s: %w", canonical, err)
	}

	if idx.docs == nil {
		idx.docs = make(map[string]document)
	}
	idx.docs[canonical] = doc
	idx.refreshMetadata()
	return nil
}

// Remove deletes the provided path from the index if present.
func (idx *Index) Remove(path string) error {
	if idx == nil {
		return nil
	}

	canonical := idx.normalize(path)
	if canonical == "" {
		return nil
	}

	if len(idx.docs) == 0 {
		return nil
	}

	delete(idx.docs, canonical)
	idx.refreshMetadata()
	return nil
}

func (idx *Index) refreshMetadata() {
	idx.aliases = idx.buildAliases()
	idx.computeRelationships()
}

func (idx *Index) normalize(path string) string {
	cleaned := filepath.Clean(path)
	if cleaned == "." || cleaned == "" {
		return ""
	}
	if filepath.IsAbs(cleaned) {
		return cleaned
	}
	joined := filepath.Join(idx.root, cleaned)
	return filepath.Clean(joined)
}

// RelatedNotes captures outbound links and backlinks for a note.
type RelatedNotes struct {
	Outbound  []string
	Backlinks []string
}

// Canonical returns the canonical path for the provided identifier if the
// document exists in the index. The method resolves aliases and normalizes the
// resulting path before verifying membership in the index.
func (idx *Index) Canonical(path string) string {
	if idx == nil {
		return ""
	}

	if resolved := idx.resolveAlias(path); resolved != "" {
		return resolved
	}

	normalized := idx.normalize(path)
	if normalized == "" {
		return ""
	}

	if _, ok := idx.docs[normalized]; ok {
		return normalized
	}
	return ""
}

// Related returns the outbound and backlink relationships for the provided
// note path. The method accepts absolute or relative paths and falls back to
// alias matching using the index metadata when possible.
func (idx *Index) Related(path string) RelatedNotes {
	canonical := filepath.Clean(path)

	// Attempt to resolve via alias lookup so relative paths or stem names
	// still succeed when called by higher level features.
	if resolved := idx.resolveAlias(canonical); resolved != "" {
		canonical = resolved
	}

	related := RelatedNotes{}
	if links, ok := idx.outbound[canonical]; ok {
		related.Outbound = append([]string(nil), links...)
	}
	if refs, ok := idx.backlinks[canonical]; ok {
		related.Backlinks = append([]string(nil), refs...)
	}
	return related
}

// Search evaluates the provided query against the index and returns matching
// note paths alongside snippets describing the match location.
func (idx *Index) Search(q Query) []Result {
	if len(idx.docs) == 0 {
		return nil
	}
	term := strings.TrimSpace(q.Term)
	loweredTerm := strings.ToLower(term)

	type scoredResult struct {
		result Result
		score  float64
	}

	now := time.Now().UTC()
	matches := make([]scoredResult, 0)
	matchedPaths := make(map[string]struct{})

	for _, doc := range idx.docs {
		if !doc.matchesFilters(q) {
			continue
		}

		if loweredTerm == "" {
			score := computeScore(doc, idx.cfg, q, 0, 0, 0, now)
			res := Result{Path: doc.Path, MatchFrom: "metadata", Score: score, Related: idx.Related(doc.Path)}
			matches = append(matches, scoredResult{result: res, score: score})
			matchedPaths[doc.Path] = struct{}{}
			continue
		}

		var snippet string
		matchFrom := ""
		totalFreq := 0
		contentMatches := 0

		if s, freq := doc.matchFrontMatter(loweredTerm); freq > 0 {
			if snippet == "" {
				snippet = s
				matchFrom = "frontmatter"
			}
			totalFreq += freq
			contentMatches++
		}

		if s, freq := doc.matchLinks(loweredTerm); freq > 0 {
			if snippet == "" {
				snippet = s
				matchFrom = "links"
			}
			totalFreq += freq
			contentMatches++
		}

		if idx.cfg.EnableBody {
			if s, freq := doc.matchBody(loweredTerm); freq > 0 {
				if snippet == "" {
					snippet = s
					matchFrom = "body"
				}
				totalFreq += freq
				contentMatches++
			}
		}

		if totalFreq > 0 {
			score := computeScore(doc, idx.cfg, q, totalFreq, contentMatches, 0, now)
			res := Result{Path: doc.Path, Snippet: snippet, MatchFrom: matchFrom, Score: score, Related: idx.Related(doc.Path)}
			matches = append(matches, scoredResult{result: res, score: score})
			matchedPaths[doc.Path] = struct{}{}
		}
	}

	if loweredTerm != "" {
		for _, doc := range idx.docs {
			if !doc.matchesFilters(q) {
				continue
			}
			if _, ok := matchedPaths[doc.Path]; ok {
				continue
			}

			candidate, similarity := doc.bestFuzzyCandidate(idx.root, loweredTerm)
			if similarity < 0.45 {
				continue
			}

			score := computeScore(doc, idx.cfg, q, 0, 1, similarity, now)
			res := Result{
				Path:      doc.Path,
				Snippet:   fmt.Sprintf("≈ %s", candidate),
				MatchFrom: "fuzzy",
				Score:     score,
				Related:   idx.Related(doc.Path),
			}
			matches = append(matches, scoredResult{result: res, score: score})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			return matches[i].result.Path < matches[j].result.Path
		}
		return matches[i].score > matches[j].score
	})

	results := make([]Result, len(matches))
	for i, m := range matches {
		results[i] = m.result
	}
	return results
}

func (idx *Index) shouldIgnore(path string) bool {
	rel, err := filepath.Rel(idx.root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	parts := strings.Split(rel, "/")
	for _, segment := range parts {
		for _, ignored := range idx.cfg.IgnoredFolders {
			if ignored == "" {
				continue
			}
			if strings.EqualFold(segment, ignored) {
				return true
			}
		}
	}
	return false
}

func (idx *Index) loadDocument(path string) (document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return document{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return document{}, err
	}

	fm, body := splitFrontMatter(data)
	parsed, tags, err := parseFrontMatter(fm)
	if err != nil {
		return document{}, fmt.Errorf("parse front matter: %w", err)
	}

	links := extractLinks(body)
	headings := extractHeadings(body)

	return document{
		Path:        filepath.Clean(path),
		Tags:        tags,
		FrontMatter: parsed,
		Links:       links,
		Body:        string(body),
		ModifiedAt:  info.ModTime().UTC(),
		Headings:    headings,
	}, nil
}

// Metadata represents the exposed metadata for an indexed document.
type Metadata struct {
	Path        string
	Tags        []string
	FrontMatter map[string][]string
	Links       []string
	ModifiedAt  time.Time
}

// Documents returns shallow copies of the metadata for indexed documents.
func (idx *Index) Documents() []Metadata {
	if len(idx.docs) == 0 {
		return nil
	}

	out := make([]Metadata, 0, len(idx.docs))
	for _, doc := range idx.docs {
		out = append(out, Metadata{
			Path:        doc.Path,
			Tags:        append([]string(nil), doc.Tags...),
			FrontMatter: cloneMetadata(doc.FrontMatter),
			Links:       append([]string(nil), doc.Links...),
			ModifiedAt:  doc.ModifiedAt,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

// FilteredDocuments returns the metadata for notes matching the provided query.
func (idx *Index) FilteredDocuments(q Query) []Metadata {
	if len(idx.docs) == 0 {
		return nil
	}

	matches := make([]Metadata, 0)
	for _, doc := range idx.docs {
		if !doc.matchesFilters(q) {
			continue
		}

		matches = append(matches, Metadata{
			Path:        doc.Path,
			Tags:        append([]string(nil), doc.Tags...),
			FrontMatter: cloneMetadata(doc.FrontMatter),
			Links:       append([]string(nil), doc.Links...),
			ModifiedAt:  doc.ModifiedAt,
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Path < matches[j].Path
	})
	return matches
}

func (idx *Index) computeRelationships() {
	outbound := make(map[string]map[string]struct{}, len(idx.docs))
	backlinks := make(map[string]map[string]struct{}, len(idx.docs))

	for path, doc := range idx.docs {
		for _, raw := range doc.Links {
			target := idx.resolveLink(path, raw)
			if target == "" || target == path {
				continue
			}

			if _, ok := outbound[path]; !ok {
				outbound[path] = make(map[string]struct{})
			}
			outbound[path][target] = struct{}{}

			if _, ok := backlinks[target]; !ok {
				backlinks[target] = make(map[string]struct{})
			}
			backlinks[target][path] = struct{}{}
		}
	}

	idx.outbound = make(map[string][]string, len(outbound))
	for path, targets := range outbound {
		idx.outbound[path] = setToSortedSlice(targets)
	}

	idx.backlinks = make(map[string][]string, len(backlinks))
	for path, sources := range backlinks {
		idx.backlinks[path] = setToSortedSlice(sources)
	}
}

func (idx *Index) buildAliases() map[string]string {
	aliases := make(map[string]string, len(idx.docs)*4)
	for path := range idx.docs {
		canonical := filepath.Clean(path)

		rel, err := filepath.Rel(idx.root, canonical)
		if err == nil {
			addAlias(aliases, rel, canonical)
		}

		addAlias(aliases, filepath.Base(canonical), canonical)
	}
	return aliases
}

func addAlias(aliases map[string]string, candidate, path string) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return
	}
	normalized := strings.ToLower(filepath.ToSlash(candidate))
	aliases[normalized] = path

	if ext := filepath.Ext(normalized); ext != "" {
		stem := strings.TrimSuffix(normalized, ext)
		if stem != "" {
			aliases[stem] = path
		}
	}
}

func (idx *Index) resolveAlias(path string) string {
	if len(idx.aliases) == 0 {
		return ""
	}
	normalized := strings.ToLower(filepath.ToSlash(path))
	if normalized == "" {
		return ""
	}
	if resolved, ok := idx.aliases[normalized]; ok {
		return resolved
	}
	if ext := filepath.Ext(normalized); ext != "" {
		stem := strings.TrimSuffix(normalized, ext)
		if resolved, ok := idx.aliases[stem]; ok {
			return resolved
		}
	}
	return ""
}

func (idx *Index) resolveLink(sourcePath, link string) string {
	if len(idx.aliases) == 0 {
		return ""
	}
	cleaned := strings.TrimSpace(link)
	if cleaned == "" {
		return ""
	}

	cleaned = strings.ReplaceAll(cleaned, "\\", "/")
	if hash := strings.Index(cleaned, "#"); hash >= 0 {
		cleaned = cleaned[:hash]
	}
	cleaned = strings.Trim(cleaned, "/")
	if cleaned == "" {
		return ""
	}

	lowered := strings.ToLower(cleaned)
	if strings.Contains(lowered, "://") || strings.HasPrefix(lowered, "mailto:") {
		return ""
	}

	if resolved := idx.resolveAlias(cleaned); resolved != "" {
		return resolved
	}

	if sourcePath != "" {
		if relative := idx.resolveRelativeLink(sourcePath, cleaned); relative != "" {
			if resolved := idx.resolveAlias(relative); resolved != "" {
				return resolved
			}
		}
	}
	return ""
}

func (idx *Index) resolveRelativeLink(sourcePath, link string) string {
	if sourcePath == "" || link == "" {
		return ""
	}

	sourceDir := filepath.Dir(sourcePath)

	joined := filepath.Join(sourceDir, link)
	if rel, err := filepath.Rel(idx.root, joined); err == nil {
		cleaned := strings.Trim(filepath.ToSlash(rel), "/")
		if cleaned != "" {
			return cleaned
		}
	}

	cleaned := strings.Trim(filepath.ToSlash(filepath.Clean(joined)), "/")
	if cleaned == "" {
		return ""
	}
	return cleaned
}

func setToSortedSlice(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for v := range values {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func cloneMetadata(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string][]string, len(values))
	for key, vals := range values {
		cloned[key] = append([]string(nil), vals...)
	}
	return cloned
}

func (d document) matchesFilters(q Query) bool {
	if len(q.Tags) > 0 {
		matched := false
		for _, required := range q.Tags {
			if containsFold(d.Tags, required) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(q.Metadata) == 0 {
		return true
	}

	for key, values := range q.Metadata {
		available, ok := d.FrontMatter[key]
		if !ok {
			return false
		}
		matched := false
		for _, want := range values {
			if containsFold(available, want) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func (d document) matchFrontMatter(term string) (string, int) {
	matches := 0
	var snippet string
	for key, values := range d.FrontMatter {
		for _, value := range values {
			lowered := strings.ToLower(value)
			if strings.Contains(lowered, term) {
				if snippet == "" {
					snippet = fmt.Sprintf("%s: %s", key, value)
				}
				matches += countOccurrences(lowered, term)
			}
		}
	}
	return snippet, matches
}

func (d document) matchLinks(term string) (string, int) {
	matches := 0
	var snippet string
	for _, link := range d.Links {
		lowered := strings.ToLower(link)
		if strings.Contains(lowered, term) {
			if snippet == "" {
				snippet = fmt.Sprintf("link: %s", link)
			}
			matches += countOccurrences(lowered, term)
		}
	}
	return snippet, matches
}

func (d document) matchBody(term string) (string, int) {
	lowered := strings.ToLower(d.Body)
	idx := strings.Index(lowered, term)
	if idx == -1 {
		return "", 0
	}
	runeStart := utf8.RuneCountInString(lowered[:idx])
	snippet := bodySnippet(d.Body, runeStart, utf8.RuneCountInString(term))
	return snippet, countOccurrences(lowered, term)
}

func (d document) bestFuzzyCandidate(root, loweredTerm string) (string, float64) {
	candidates := make([]string, 0, 4+len(d.Headings))

	if rel, err := filepath.Rel(root, d.Path); err == nil {
		rel = filepath.ToSlash(rel)
		if rel != "" {
			candidates = append(candidates, rel)
		}
	}

	base := filepath.Base(d.Path)
	if base != "" {
		candidates = append(candidates, base)
		if ext := filepath.Ext(base); ext != "" {
			stem := strings.TrimSuffix(base, ext)
			if stem != "" {
				candidates = append(candidates, stem)
			}
		}
	}

	candidates = append(candidates, d.Headings...)

	for _, values := range d.FrontMatter {
		candidates = append(candidates, values...)
	}

	seen := make(map[string]struct{}, len(candidates))
	bestCandidate := ""
	bestScore := 0.0

	for _, candidate := range candidates {
		cleaned := strings.TrimSpace(candidate)
		if cleaned == "" {
			continue
		}
		lowered := strings.ToLower(cleaned)
		if _, ok := seen[lowered]; ok {
			continue
		}
		seen[lowered] = struct{}{}

		score := similarityScore(loweredTerm, lowered)
		if score > bestScore {
			bestScore = score
			bestCandidate = cleaned
		}
	}

	return bestCandidate, bestScore
}

func computeScore(doc document, cfg Config, q Query, termFreq int, contentMatches int, similarity float64, now time.Time) float64 {
	const (
		recencyWeight   = 0.45
		frequencyWeight = 0.35
		tagWeight       = 0.15
		contextWeight   = 0.05
	)

	recencyScore := recencyComponent(doc.ModifiedAt, now)
	frequencyScore := frequencyComponent(termFreq, similarity)
	tagScore, tagMatches := tagOverlapScore(doc.Tags, q.Tags)
	metadataMatches := metadataMatchCount(doc, q)

	contentTotal := 0
	if termFreq > 0 || contentMatches > 0 {
		contentTotal = 2
		if cfg.EnableBody {
			contentTotal++
		}
	} else if similarity > 0 {
		contentTotal = 1
	}

	totalContexts := contentTotal + len(q.Tags) + len(q.Metadata)
	if totalContexts == 0 {
		totalContexts = 1
	}
	matchedContexts := contentMatches + tagMatches + metadataMatches
	contextScore := float64(matchedContexts) / float64(totalContexts)

	return recencyWeight*recencyScore + frequencyWeight*frequencyScore + tagWeight*tagScore + contextWeight*contextScore
}

func recencyComponent(modifiedAt, now time.Time) float64 {
	if modifiedAt.IsZero() {
		return 0
	}
	age := now.Sub(modifiedAt)
	if age < 0 {
		age = 0
	}
	const decay = 30 * 24 * time.Hour
	if decay <= 0 {
		return 0
	}
	return math.Exp(-age.Hours() / decay.Hours())
}

func frequencyComponent(freq int, similarity float64) float64 {
	if freq < 0 {
		freq = 0
	}
	tf := 0.0
	if freq > 0 {
		tf = 1 - math.Exp(-float64(freq))
	}
	if similarity > tf {
		tf = similarity
	}
	if tf > 1 {
		tf = 1
	}
	return tf
}

func tagOverlapScore(tags, required []string) (float64, int) {
	if len(required) == 0 {
		return 0, 0
	}
	matches := 0
	for _, want := range required {
		if containsFold(tags, want) {
			matches++
		}
	}
	return float64(matches) / float64(len(required)), matches
}

func metadataMatchCount(doc document, q Query) int {
	if len(q.Metadata) == 0 {
		return 0
	}
	matches := 0
	for key, values := range q.Metadata {
		available, ok := doc.FrontMatter[key]
		if !ok {
			continue
		}
		for _, want := range values {
			if containsFold(available, want) {
				matches++
				break
			}
		}
	}
	return matches
}

func countOccurrences(haystack, needle string) int {
	if haystack == "" || needle == "" {
		return 0
	}
	return strings.Count(haystack, needle)
}

func extractHeadings(content []byte) []string {
	lines := strings.Split(string(content), "\n")
	headings := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		trimmed = strings.TrimLeft(trimmed, "#")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" {
			headings = append(headings, trimmed)
		}
	}
	return headings
}

func similarityScore(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}

	ar := []rune(a)
	br := []rune(b)
	if len(ar) < 3 || len(br) < 3 {
		distance := levenshteinDistance(ar, br)
		maxLen := max(len(ar), len(br))
		if maxLen == 0 {
			return 1
		}
		return 1 - float64(distance)/float64(maxLen)
	}

	return trigramSimilarity(a, b)
}

func trigramSimilarity(a, b string) float64 {
	aTrigrams := toTrigramSet(a)
	bTrigrams := toTrigramSet(b)
	if len(aTrigrams) == 0 || len(bTrigrams) == 0 {
		return 0
	}

	intersection := 0
	for tri := range aTrigrams {
		if _, ok := bTrigrams[tri]; ok {
			intersection++
		}
	}

	return float64(2*intersection) / float64(len(aTrigrams)+len(bTrigrams))
}

func toTrigramSet(s string) map[string]struct{} {
	trigrams := make(map[string]struct{})
	for i := 0; i <= len(s)-3; i++ {
		tri := s[i : i+3]
		trigrams[tri] = struct{}{}
	}
	return trigrams
}

func levenshteinDistance(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	dp := make([]int, len(b)+1)
	for j := range dp {
		dp[j] = j
	}

	for i := 1; i <= len(a); i++ {
		prev := dp[0]
		dp[0] = i
		for j := 1; j <= len(b); j++ {
			temp := dp[j]
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			insertion := dp[j] + 1
			deletion := dp[j-1] + 1
			substitution := prev + cost

			dp[j] = minInt(insertion, deletion, substitution)
			prev = temp
		}
	}

	return dp[len(b)]
}

func minInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func containsFold(values []string, target string) bool {
	for _, v := range values {
		if strings.EqualFold(v, target) {
			return true
		}
	}
	return false
}

func splitFrontMatter(data []byte) ([]byte, []byte) {
	re := regexp.MustCompile(`(?ms)^---\s*\n(.*?)\n---\s*\n?`)
	loc := re.FindSubmatchIndex(data)
	if len(loc) < 4 {
		return nil, data
	}
	yamlStart := loc[2]
	yamlEnd := loc[3]
	bodyStart := loc[1]
	fm := data[yamlStart:yamlEnd]
	body := data[bodyStart:]
	return fm, body
}

func parseFrontMatter(fm []byte) (map[string][]string, []string, error) {
	result := make(map[string][]string)
	var tags []string
	if len(fm) == 0 {
		return result, tags, nil
	}

	var data yaml.Node
	if err := yaml.Unmarshal(fm, &data); err != nil {
		return nil, nil, err
	}

	if data.Kind != yaml.DocumentNode || len(data.Content) == 0 {
		return result, tags, nil
	}

	mapping := data.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return result, tags, nil
	}

	for i := 0; i < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valueNode := mapping.Content[i+1]
		key := keyNode.Value

		values := flattenYAMLValue(valueNode)
		result[key] = values
		if key == "tags" {
			tags = values
		}
	}

	return result, tags, nil
}

func flattenYAMLValue(node *yaml.Node) []string {
	switch node.Kind {
	case yaml.SequenceNode:
		vals := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			vals = append(vals, child.Value)
		}
		return vals
	case yaml.ScalarNode:
		return []string{node.Value}
	default:
		return nil
	}
}

func extractLinks(content []byte) []string {
	body := string(content)
	links := make(map[string]struct{})

	wikiRe := regexp.MustCompile(`\[\[(.+?)\]\]`)
	for _, match := range wikiRe.FindAllStringSubmatch(body, -1) {
		if len(match) > 1 {
			links[strings.TrimSpace(match[1])] = struct{}{}
		}
	}

	mdRe := regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	for _, match := range mdRe.FindAllStringSubmatch(body, -1) {
		if len(match) > 1 {
			links[strings.TrimSpace(match[1])] = struct{}{}
		}
	}

	out := make([]string, 0, len(links))
	for link := range links {
		out = append(out, link)
	}

	sort.Strings(out)
	return out
}

func bodySnippet(body string, index, termLen int) string {
	if termLen <= 0 {
		termLen = 1
	}

	runes := []rune(body)
	start := index
	end := index + termLen
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}

	const window = 40
	snippetStart := max(0, start-window)
	snippetEnd := min(len(runes), end+window)

	snippet := string(runes[snippetStart:snippetEnd])
	snippet = strings.TrimSpace(snippet)
	if snippetStart > 0 {
		snippet = "…" + snippet
	}
	if snippetEnd < len(runes) {
		snippet = snippet + "…"
	}
	return snippet
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
