package review

import "github.com/Paintersrp/an/internal/search"

// GraphNode represents backlinks and outbound connections for a note.
type GraphNode struct {
	Path      string
	Outbound  []string
	Backlinks []string
}

// Graph captures a backlink graph for the provided seeds.
type Graph struct {
	Nodes map[string]GraphNode
}

// BuildBacklinkGraph constructs a backlink graph for the provided seed paths.
// The resulting graph includes seed nodes and their direct neighbors.
func BuildBacklinkGraph(idx *search.Index, seeds []string) Graph {
	graph := Graph{Nodes: make(map[string]GraphNode)}
	if idx == nil {
		return graph
	}

	visited := make(map[string]struct{})
	for _, seed := range seeds {
		canonical := idx.Canonical(seed)
		if canonical == "" {
			continue
		}
		if _, ok := visited[canonical]; ok {
			continue
		}
		visited[canonical] = struct{}{}

		related := idx.Related(canonical)
		node := GraphNode{
			Path:      canonical,
			Outbound:  append([]string(nil), related.Outbound...),
			Backlinks: append([]string(nil), related.Backlinks...),
		}
		graph.Nodes[canonical] = node

		for _, neighbor := range append(append([]string(nil), node.Outbound...), node.Backlinks...) {
			normalizedNeighbor := idx.Canonical(neighbor)
			if normalizedNeighbor == "" {
				continue
			}
			if _, ok := graph.Nodes[normalizedNeighbor]; ok {
				continue
			}
			relatedNeighbor := idx.Related(normalizedNeighbor)
			graph.Nodes[normalizedNeighbor] = GraphNode{
				Path:      normalizedNeighbor,
				Outbound:  append([]string(nil), relatedNeighbor.Outbound...),
				Backlinks: append([]string(nil), relatedNeighbor.Backlinks...),
			}
		}
	}

	return graph
}
