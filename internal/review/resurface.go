package review

import (
	"sort"
	"strings"
	"time"

	"github.com/Paintersrp/an/internal/search"
)

// Bucket represents a resurfacing cadence bucket.
type Bucket struct {
	// Name is the human readable label for the bucket (for example "weekly").
	Name string
	// After specifies the minimum age a note must reach before falling into the
	// bucket. Buckets are evaluated from the smallest After duration to the
	// largest, with the last matching bucket winning.
	After time.Duration
}

// ResurfaceOptions configures queue generation.
type ResurfaceOptions struct {
	Now        time.Time
	MinimumAge time.Duration
	Limit      int
	Buckets    []Bucket
	Query      search.Query
}

// ResurfaceItem captures queue metadata for a resurfaced note.
type ResurfaceItem struct {
	Path       string
	Tags       []string
	Metadata   map[string][]string
	Links      []string
	ModifiedAt time.Time
	Age        time.Duration
	Bucket     string
}

// DefaultBuckets returns the default resurfacing cadence buckets.
func DefaultBuckets() []Bucket {
	return []Bucket{
		{Name: "daily", After: 24 * time.Hour},
		{Name: "every-3-days", After: 72 * time.Hour},
		{Name: "weekly", After: 7 * 24 * time.Hour},
		{Name: "biweekly", After: 14 * 24 * time.Hour},
		{Name: "monthly", After: 30 * 24 * time.Hour},
	}
}

// BuildResurfaceQueue evaluates the index metadata and returns the resurfacing
// queue ordered from stalest to freshest.
func BuildResurfaceQueue(idx *search.Index, opts ResurfaceOptions) []ResurfaceItem {
	if idx == nil {
		return nil
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	buckets := append([]Bucket(nil), opts.Buckets...)
	if len(buckets) == 0 {
		buckets = DefaultBuckets()
	}
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].After < buckets[j].After
	})

	docs := idx.FilteredDocuments(opts.Query)
	if len(docs) == 0 {
		return nil
	}

	items := make([]ResurfaceItem, 0, len(docs))
	for _, doc := range docs {
		age := now.Sub(doc.ModifiedAt)
		if opts.MinimumAge > 0 && age < opts.MinimumAge {
			continue
		}

		bucket := determineBucket(age, buckets)
		if bucket == "" {
			continue
		}

		items = append(items, ResurfaceItem{
			Path:       doc.Path,
			Tags:       append([]string(nil), doc.Tags...),
			Metadata:   cloneMetadata(doc.FrontMatter),
			Links:      append([]string(nil), doc.Links...),
			ModifiedAt: doc.ModifiedAt,
			Age:        age,
			Bucket:     bucket,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Bucket != items[j].Bucket {
			return bucketRank(items[i].Bucket, buckets) > bucketRank(items[j].Bucket, buckets)
		}
		if !items[i].ModifiedAt.Equal(items[j].ModifiedAt) {
			return items[i].ModifiedAt.Before(items[j].ModifiedAt)
		}
		return items[i].Path < items[j].Path
	})

	if opts.Limit > 0 && len(items) > opts.Limit {
		items = items[:opts.Limit]
	}
	return items
}

func determineBucket(age time.Duration, buckets []Bucket) string {
	matched := ""
	for _, bucket := range buckets {
		if age >= bucket.After {
			matched = bucket.Name
		}
	}
	return matched
}

func bucketRank(name string, buckets []Bucket) int {
	for idx, bucket := range buckets {
		if bucket.Name == name {
			return idx
		}
	}
	return len(buckets)
}

func cloneMetadata(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string][]string, len(values))
	for key, vals := range values {
		copied[key] = append([]string(nil), vals...)
	}
	return copied
}

// FilterQueue filters resurfacing items by tags and metadata, returning the
// subset that satisfies all requested filters.
func FilterQueue(items []ResurfaceItem, tags []string, metadata map[string][]string) []ResurfaceItem {
	if len(tags) == 0 && len(metadata) == 0 {
		return items
	}

	filtered := make([]ResurfaceItem, 0, len(items))
	for _, item := range items {
		if len(tags) > 0 && !containsAllTags(item.Tags, tags) {
			continue
		}
		if len(metadata) > 0 && !matchesMetadata(item.Metadata, metadata) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func containsAllTags(tags, required []string) bool {
	if len(required) == 0 {
		return true
	}

	lookup := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		lookup[strings.ToLower(tag)] = struct{}{}
	}

	for _, want := range required {
		if _, ok := lookup[strings.ToLower(want)]; !ok {
			return false
		}
	}
	return true
}

func matchesMetadata(values map[string][]string, required map[string][]string) bool {
	for key, wantValues := range required {
		available, ok := values[key]
		if !ok {
			return false
		}

		wantLookup := make(map[string]struct{}, len(wantValues))
		for _, want := range wantValues {
			wantLookup[strings.ToLower(want)] = struct{}{}
		}

		for _, have := range available {
			lowered := strings.ToLower(have)
			delete(wantLookup, lowered)
		}

		if len(wantLookup) > 0 {
			return false
		}
	}
	return true
}
