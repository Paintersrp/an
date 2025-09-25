package notes

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Paintersrp/an/internal/pathutil"
	"github.com/Paintersrp/an/internal/review"
	"github.com/Paintersrp/an/internal/search"
)

type previewContext struct {
	Path            string
	Outbound        []string
	Backlinks       []string
	QueueNeighbours []string
}

const maxContextItems = 8

func buildPreviewContext(
	path string,
	idx *search.Index,
	queue []string,
	override *search.RelatedNotes,
) previewContext {
	ctx := previewContext{}
	canonical := canonicalPath(idx, path)
	ctx.Path = canonical

	if override != nil {
		ctx.Outbound = copyStrings(override.Outbound)
		ctx.Backlinks = copyStrings(override.Backlinks)
	} else if idx != nil {
		related := idx.Related(canonical)
		ctx.Outbound = copyStrings(related.Outbound)
		ctx.Backlinks = copyStrings(related.Backlinks)
	}

	if len(queue) > 0 && idx != nil {
		seeds, queueSet := canonicalQueue(queue, idx)
		if len(seeds) > 0 {
			graph := review.BuildBacklinkGraph(idx, seeds)
			matches := queueNeighbours(graph, canonical, queueSet)
			if len(matches) > 0 {
				ctx.QueueNeighbours = matches
			}
		}
	}

	return ctx
}

func formatPreviewContext(ctx previewContext, vault string) string {
	summary := fmt.Sprintf(
		"Links: %d outbound · %d backlinks",
		len(ctx.Outbound),
		len(ctx.Backlinks),
	)
	if len(ctx.QueueNeighbours) > 0 {
		summary = fmt.Sprintf(
			"%s · %d queue neighbours",
			summary,
			len(ctx.QueueNeighbours),
		)
	}

	sections := []struct {
		title string
		items []string
	}{
		{title: "Outbound", items: ctx.Outbound},
		{title: "Backlinks", items: ctx.Backlinks},
	}
	if len(ctx.QueueNeighbours) > 0 {
		sections = append(sections, struct {
			title string
			items []string
		}{title: "Queue neighbours", items: ctx.QueueNeighbours})
	}

	var builder strings.Builder
	builder.WriteString(summary)

	for _, section := range sections {
		if len(section.items) == 0 {
			continue
		}

		builder.WriteString("\n")
		builder.WriteString(section.title)
		builder.WriteString(":\n")

		display := displayPaths(section.items, vault)
		shown, hidden := limitItems(display, len(section.items), maxContextItems)
		for _, item := range shown {
			builder.WriteString("  • ")
			builder.WriteString(item)
			builder.WriteString("\n")
		}
		if hidden > 0 {
			builder.WriteString(fmt.Sprintf("  • … and %d more\n", hidden))
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}

func canonicalPath(idx *search.Index, path string) string {
	cleaned := pathutil.NormalizePath(path)
	if idx == nil {
		return cleaned
	}
	if resolved := idx.Canonical(cleaned); resolved != "" {
		return resolved
	}
	return cleaned
}

func canonicalQueue(queue []string, idx *search.Index) ([]string, map[string]struct{}) {
	if len(queue) == 0 {
		return nil, nil
	}

	set := make(map[string]struct{}, len(queue))
	seeds := make([]string, 0, len(queue))
	for _, candidate := range queue {
		cleaned := canonicalPath(idx, candidate)
		if cleaned == "" {
			continue
		}
		if _, ok := set[cleaned]; ok {
			continue
		}
		set[cleaned] = struct{}{}
		seeds = append(seeds, cleaned)
	}
	sort.Strings(seeds)
	return seeds, set
}

func queueNeighbours(
	graph review.Graph,
	path string,
	queueSet map[string]struct{},
) []string {
	if len(graph.Nodes) == 0 || len(queueSet) == 0 {
		return nil
	}

	node, ok := graph.Nodes[path]
	if !ok {
		return nil
	}

	matches := make(map[string]struct{})
	for _, candidate := range append(copyStrings(node.Outbound), node.Backlinks...) {
		if _, ok := queueSet[candidate]; ok {
			matches[candidate] = struct{}{}
		}
	}
	if len(matches) == 0 {
		return nil
	}

	out := make([]string, 0, len(matches))
	for candidate := range matches {
		out = append(out, candidate)
	}
	sort.Strings(out)
	return out
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func limitItems(items []string, originalCount, limit int) ([]string, int) {
	if limit <= 0 || len(items) <= limit {
		return copyStrings(items), originalCount - len(items)
	}
	shown := make([]string, limit)
	copy(shown, items[:limit])
	return shown, originalCount - limit
}

func displayPaths(paths []string, vault string) []string {
	if len(paths) == 0 {
		return nil
	}

	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, displayPath(p, vault))
	}
	return out
}

func displayPath(path, vault string) string {
	cleaned := pathutil.NormalizePath(path)
	if vault != "" {
		if rel, err := pathutil.VaultRelative(vault, cleaned); err == nil {
			rel = strings.TrimPrefix(rel, "./")
			if rel != "" && rel != "." {
				return rel
			}
		}
	}
	base := filepath.Base(cleaned)
	return filepath.ToSlash(base)
}
