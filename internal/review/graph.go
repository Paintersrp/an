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
		if _, ok := visited[seed]; ok {
			continue
		}
		visited[seed] = struct{}{}

		related := idx.Related(seed)
		node := GraphNode{
			Path:      seed,
			Outbound:  append([]string(nil), related.Outbound...),
			Backlinks: append([]string(nil), related.Backlinks...),
		}
		graph.Nodes[seed] = node

		for _, neighbor := range append(append([]string(nil), node.Outbound...), node.Backlinks...) {
			if _, ok := graph.Nodes[neighbor]; ok {
				continue
			}
			relatedNeighbor := idx.Related(neighbor)
			graph.Nodes[neighbor] = GraphNode{
				Path:      neighbor,
				Outbound:  append([]string(nil), relatedNeighbor.Outbound...),
				Backlinks: append([]string(nil), relatedNeighbor.Backlinks...),
			}
		}
	}

	return graph
}
