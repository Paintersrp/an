package search

import (
	"errors"
	"fmt"
	"io/fs"
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

	results := make([]Result, 0)
	for _, doc := range idx.docs {
		if !doc.matchesFilters(q) {
			continue
		}

		if loweredTerm == "" {
			// Pure metadata filtering request.
			results = append(results, Result{Path: doc.Path, MatchFrom: "metadata"})
			continue
		}

		if snippet, ok := doc.matchFrontMatter(loweredTerm); ok {
			results = append(results, Result{Path: doc.Path, Snippet: snippet, MatchFrom: "frontmatter"})
			continue
		}

		if snippet, ok := doc.matchLinks(loweredTerm); ok {
			results = append(results, Result{Path: doc.Path, Snippet: snippet, MatchFrom: "links"})
			continue
		}

		if idx.cfg.EnableBody {
			if snippet, ok := doc.matchBody(loweredTerm); ok {
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

	return document{
		Path:        filepath.Clean(path),
		Tags:        tags,
		FrontMatter: parsed,
		Links:       extractLinks(body),
		Body:        string(body),
		ModifiedAt:  info.ModTime().UTC(),
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
		for _, required := range q.Tags {
			if !containsFold(d.Tags, required) {
				return false
			}
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
		for _, want := range values {
			if !containsFold(available, want) {
				return false
			}
		}
	}
	return true
}

func (d document) matchFrontMatter(term string) (string, bool) {
	for key, values := range d.FrontMatter {
		for _, value := range values {
			if strings.Contains(strings.ToLower(value), term) {
				return fmt.Sprintf("%s: %s", key, value), true
			}
		}
	}
	return "", false
}

func (d document) matchLinks(term string) (string, bool) {
	for _, link := range d.Links {
		if strings.Contains(strings.ToLower(link), term) {
			return fmt.Sprintf("link: %s", link), true
		}
	}
	return "", false
}

func (d document) matchBody(term string) (string, bool) {
	lowered := strings.ToLower(d.Body)
	idx := strings.Index(lowered, term)
	if idx == -1 {
		return "", false
	}
	runeStart := utf8.RuneCountInString(lowered[:idx])
	return bodySnippet(d.Body, runeStart, utf8.RuneCountInString(term)), true
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
