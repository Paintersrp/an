package add

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/views"
)

func NewCmdViewAdd(s *state.State) *cobra.Command {
	var (
		name       string
		include    []string
		exclude    []string
		sortField  string
		sortOrder  string
		predicates []string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a custom view",
		RunE: func(cmd *cobra.Command, args []string) error {
			trimmedName := strings.TrimSpace(name)
			if trimmedName == "" {
				return fmt.Errorf("view name is required")
			}

			field := strings.ToLower(strings.TrimSpace(sortField))
			if field == "" {
				field = string(views.SortFieldModified)
			}
			if !views.IsValidSortField(views.SortField(field)) {
				return fmt.Errorf("invalid sort field: %s", sortField)
			}

			order := strings.ToLower(strings.TrimSpace(sortOrder))
			if order == "" {
				order = string(views.SortOrderDescending)
			}
			if !views.IsValidSortOrder(views.SortOrder(order)) {
				return fmt.Errorf("invalid sort order: %s", sortOrder)
			}

			normalizedPredicates := normalizeSlice(predicates)
			for i, predicate := range normalizedPredicates {
				normalized := strings.ToLower(predicate)
				if !views.IsValidPredicate(views.Predicate(normalized)) {
					return fmt.Errorf("invalid predicate: %s", predicate)
				}
				normalizedPredicates[i] = normalized
			}

			def := config.ViewDefinition{
				Include:    normalizeSlice(include),
				Exclude:    normalizeSlice(exclude),
				Sort:       config.ViewSort{Field: field, Order: order},
				Predicates: normalizedPredicates,
			}

			if err := s.ViewManager.AddCustomView(trimmedName, def); err != nil {
				return err
			}

			cmd.Printf("Added view %q\n", trimmedName)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of the view to add")
	cmd.Flags().StringSliceVar(&include, "include", nil, "Include patterns for the view")
	cmd.Flags().StringSliceVar(&exclude, "exclude", nil, "Exclude patterns for the view")
	cmd.Flags().StringVar(&sortField, "sort-field", string(views.SortFieldModified), "Default sort field (title, subdirectory, modified)")
	cmd.Flags().StringVar(&sortOrder, "sort-order", string(views.SortOrderDescending), "Default sort order (asc, desc)")
	cmd.Flags().StringSliceVar(&predicates, "predicate", nil, "Predicates to apply (orphan, unfulfilled)")

	cmd.MarkFlagRequired("name")

	return cmd
}

func normalizeSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}
