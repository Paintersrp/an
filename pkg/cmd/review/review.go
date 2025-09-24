package review

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	reviewsvc "github.com/Paintersrp/an/internal/review"
	"github.com/Paintersrp/an/internal/search"
	"github.com/Paintersrp/an/internal/state"
)

// metadataFlag captures repeated key=value metadata filters.
type metadataFlag map[string][]string

func (m metadataFlag) String() string {
	if len(m) == 0 {
		return ""
	}
	parts := make([]string, 0, len(m))
	for key, values := range m {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(values, ",")))
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

func (m metadataFlag) Type() string {
	return "key=value"
}

func (m metadataFlag) Set(value string) error {
	if value == "" {
		return nil
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("metadata must be key=value, got %q", value)
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return errors.New("metadata key cannot be empty")
	}
	values := strings.Split(parts[1], ",")
	cleaned := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	if len(cleaned) == 0 {
		return fmt.Errorf("metadata %q has no values", key)
	}
	m[key] = append(m[key], cleaned...)
	return nil
}

// NewCmdReview wires the `review` command that orchestrates resurfacing,
// backlink visualization, and guided checklists.
func NewCmdReview(s *state.State) *cobra.Command {
	var (
		mode      string
		limit     int
		minAge    time.Duration
		showGraph bool
		tags      []string
		meta      = make(metadataFlag)
	)

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Run guided review rituals backed by resurfacing queues",
		Long: `Review assembles the notes that deserve another look,
surfaces how they connect, and walks you through a repeatable checklist.
Use it to keep daily, weekly, or project retrospectives inside your vault.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s == nil {
				return errors.New("state is not configured")
			}

			ws := s.Config.MustWorkspace()
			searchCfg := search.Config{
				EnableBody:     ws.Search.EnableBody,
				IgnoredFolders: append([]string(nil), ws.Search.IgnoredFolders...),
			}

			query := search.Query{
				Tags:     append([]string(nil), ws.Search.DefaultTagFilters...),
				Metadata: cloneMetadata(ws.Search.DefaultMetadataFilters),
			}
			query.Tags = append(query.Tags, tags...)
			for key, values := range meta {
				query.Metadata[key] = append(query.Metadata[key], values...)
			}

			paths, err := collectNotePaths(s.Vault, searchCfg.IgnoredFolders)
			if err != nil {
				return err
			}

			idx := search.NewIndex(s.Vault, searchCfg)
			if err := idx.Build(paths); err != nil {
				return fmt.Errorf("build search index: %w", err)
			}

			queue := reviewsvc.BuildResurfaceQueue(idx, reviewsvc.ResurfaceOptions{
				Now:        time.Now(),
				MinimumAge: minAge,
				Limit:      limit,
				Buckets:    reviewsvc.DefaultBuckets(),
				Query:      query,
			})

			out := cmd.OutOrStdout()
			if len(queue) == 0 {
				fmt.Fprintln(out, "No notes are due for resurfacing.")
			} else {
				fmt.Fprintf(out, "Resurfacing queue (%d items):\n", len(queue))
				for i, item := range queue {
					fmt.Fprintf(
						out,
						"%2d. %s — last touched %s (%s)\n",
						i+1,
						relPath(s.Vault, item.Path),
						humanizeAge(item.Age),
						item.Bucket,
					)
				}
			}

			if showGraph && len(queue) > 0 {
				fmt.Fprintln(out, "\nBacklink graph:")
				seedPaths := make([]string, 0, len(queue))
				for _, item := range queue {
					seedPaths = append(seedPaths, item.Path)
				}
				graph := reviewsvc.BuildBacklinkGraph(idx, seedPaths)
				printGraph(out, graph, s.Vault)
			}

			templateName, err := resolveTemplate(mode)
			if err != nil {
				return err
			}

			manifest, err := s.Templater.Manifest(templateName)
			if err != nil {
				return fmt.Errorf("load %s manifest: %w", templateName, err)
			}

			_, err = reviewsvc.RunChecklist(
				manifest,
				queue,
				cmd.InOrStdin(),
				out,
			)
			return err
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "daily", "Review mode to run (daily, weekly, retro)")
	cmd.Flags().IntVar(&limit, "limit", 12, "Maximum resurfacing candidates to include")
	cmd.Flags().DurationVar(&minAge, "min-age", 0, "Minimum age (for example 48h) before resurfacing a note")
	cmd.Flags().BoolVar(&showGraph, "graph", false, "Render a backlink graph for resurfacing candidates")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Additional tag filters to apply to the resurfacing queue")
	cmd.Flags().Var(meta, "metadata", "Metadata filter in key=value form (repeatable)")

	return cmd
}

func collectNotePaths(root string, ignored []string) ([]string, error) {
	normalized := make(map[string]struct{}, len(ignored))
	for _, dir := range ignored {
		normalized[strings.ToLower(dir)] = struct{}{}
	}

	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			if _, skip := normalized[name]; skip {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) == ".md" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(paths)
	return paths, nil
}

func cloneMetadata(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return make(map[string][]string)
	}
	cloned := make(map[string][]string, len(values))
	for key, vals := range values {
		cloned[key] = append([]string(nil), vals...)
	}
	return cloned
}

func resolveTemplate(mode string) (string, error) {
	switch strings.ToLower(mode) {
	case "daily":
		return "review-daily", nil
	case "weekly":
		return "review-weekly", nil
	case "retro", "project", "project-retro":
		return "review-retro", nil
	default:
		return "", fmt.Errorf("unknown review mode %q", mode)
	}
}

func humanizeAge(age time.Duration) string {
	if age < time.Hour {
		minutes := int(age.Minutes())
		if minutes <= 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	days := int(age.Hours() / 24)
	if days == 0 {
		hours := int(age.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func printGraph(out io.Writer, graph reviewsvc.Graph, root string) {
	if out == nil {
		return
	}

	paths := make([]string, 0, len(graph.Nodes))
	for path := range graph.Nodes {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		node := graph.Nodes[path]
		fmt.Fprintf(out, "- %s\n", relPath(root, node.Path))
		if len(node.Backlinks) > 0 {
			fmt.Fprintf(out, "    ← %s\n", joinPaths(root, node.Backlinks))
		}
		if len(node.Outbound) > 0 {
			fmt.Fprintf(out, "    → %s\n", joinPaths(root, node.Outbound))
		}
	}
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func joinPaths(root string, paths []string) string {
	converted := make([]string, 0, len(paths))
	for _, p := range paths {
		converted = append(converted, relPath(root, p))
	}
	return strings.Join(converted, ", ")
}
