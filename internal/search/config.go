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
	// Tags enumerates tag names where a document matches when it includes at
	// least one of the provided values.
	Tags []string
	// Metadata filters require front-matter fields to contain one or more
	// values. The values associated with a key are treated as an OR match,
	// while different keys must all be satisfied.
	Metadata map[string][]string
}

// Result captures a document match from the index.
type Result struct {
	Path      string
	Snippet   string
	MatchFrom string
	Score     float64
	Related   RelatedNotes
}
