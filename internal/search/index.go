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
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

type document struct {
	Path        string
	Tags        []string
	FrontMatter map[string][]string
	Links       []string
	Body        string
}

// Index stores searchable representations of notes on disk.
type Index struct {
	root string
	cfg  Config
	docs map[string]document
}

// NewIndex constructs an empty index rooted at the provided directory.
func NewIndex(root string, cfg Config) *Index {
	return &Index{
		root: filepath.Clean(root),
		cfg:  cfg,
		docs: make(map[string]document),
	}
}

// Build replaces the index contents using the provided note paths.
func (idx *Index) Build(paths []string) error {
	idx.docs = make(map[string]document, len(paths))
	for _, p := range paths {
		if idx.shouldIgnore(p) {
			continue
		}

		doc, err := idx.loadDocument(p)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("search: indexing %s: %w", p, err)
		}
		idx.docs[filepath.Clean(p)] = doc
	}
	return nil
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
	}, nil
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
