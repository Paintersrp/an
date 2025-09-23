package search

// Config describes index behavior.
type Config struct {
	// EnableBody controls whether the index searches note bodies in addition to
	// front matter and links.
	EnableBody bool
	// IgnoredFolders contains directory names that should be skipped when
	// indexing. Paths containing any of these folders will not be indexed.
	IgnoredFolders []string
}

// Query represents a search request against the index.
type Query struct {
	// Term is the free-text query to evaluate against indexed content.
	Term string
	// Tags enumerates tag names that must be present on the note. All tags must
	// be satisfied for a document to match.
	Tags []string
	// Metadata filters require front-matter fields to contain one or more
	// values. The values associated with a key are treated as an AND match.
	Metadata map[string][]string
}

// Result captures a document match from the index.
type Result struct {
	Path      string
	Snippet   string
	MatchFrom string
}
