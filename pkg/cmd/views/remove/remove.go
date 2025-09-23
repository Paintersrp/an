package remove

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
)

func NewCmdViewRemove(s *state.State) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a custom view",
		RunE: func(cmd *cobra.Command, args []string) error {
			trimmed := strings.TrimSpace(name)
			if trimmed == "" {
				return fmt.Errorf("view name is required")
			}

			if err := s.ViewManager.RemoveCustomView(trimmed); err != nil {
				return err
			}

			cmd.Printf("Removed view %q\n", trimmed)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of the view to remove")
	cmd.MarkFlagRequired("name")

	return cmd
}
