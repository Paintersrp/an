package workspace

import (
	"fmt"
	"maps"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
)

func NewCmdWorkspace(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage workspaces",
	}

	cmd.AddCommand(
		newCmdWorkspaceList(s),
		newCmdWorkspaceSwitch(s),
		newCmdWorkspaceAdd(s),
		newCmdWorkspaceRemove(s),
	)

	return cmd
}

func newCmdWorkspaceList(s *state.State) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured workspaces",
		RunE: func(cmd *cobra.Command, _ []string) error {
			names := s.Config.WorkspaceNames()
			if len(names) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No workspaces configured")
				return nil
			}

			for _, name := range names {
				marker := " "
				if name == s.Config.CurrentWorkspace {
					marker = "*"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", marker, name)
			}

			return nil
		},
	}
}

func newCmdWorkspaceSwitch(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch [name]",
		Short: "Switch the active workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := strings.TrimSpace(args[0])
			if target == "" {
				return fmt.Errorf("workspace name cannot be empty")
			}

			if err := s.Config.SwitchWorkspace(target); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Switched to workspace %q\n", target)
			return nil
		},
	}
	return cmd
}

func newCmdWorkspaceAdd(s *state.State) *cobra.Command {
	var name string
	var vault string
	var makeCurrent bool

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("workspace name is required")
			}
			vault = strings.TrimSpace(vault)
			if vault == "" {
				return fmt.Errorf("vault path is required")
			}

			template := cloneWorkspaceSettings(s.Workspace)
			template.VaultDir = vault
			ws := template

			if err := s.Config.AddWorkspace(name, ws, makeCurrent); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Added workspace %q\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of the new workspace")
	cmd.Flags().StringVar(&vault, "vault", "", "Path to the workspace vault")
	cmd.Flags().BoolVar(&makeCurrent, "current", false, "Switch to the new workspace after creation")

	return cmd
}

func newCmdWorkspaceRemove(s *state.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [name]",
		Short: "Remove an existing workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("workspace name cannot be empty")
			}

			if err := s.Config.RemoveWorkspace(name); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Removed workspace %q\n", name)
			return nil
		},
	}

	return cmd
}

func cloneWorkspaceSettings(src *config.Workspace) *config.Workspace {
	if src == nil {
		return &config.Workspace{}
	}

	clone := &config.Workspace{
		Editor:         src.Editor,
		NvimArgs:       src.NvimArgs,
		FileSystemMode: src.FileSystemMode,
		PinnedFile:     src.PinnedFile,
		PinnedTaskFile: src.PinnedTaskFile,
		SubDirs:        append([]string(nil), src.SubDirs...),
		Search:         src.Search,
		Views:          maps.Clone(src.Views),
		ViewOrder:      append([]string(nil), src.ViewOrder...),
		EditorTemplate: cloneCommandTemplate(src.EditorTemplate),
		Hooks: config.HookConfig{
			PreOpen:    cloneHookCommands(src.Hooks.PreOpen),
			PostOpen:   cloneHookCommands(src.Hooks.PostOpen),
			PostCreate: cloneHookCommands(src.Hooks.PostCreate),
		},
	}
	clone.Search.DefaultMetadataFilters = map[string][]string{}
	for key, values := range src.Search.DefaultMetadataFilters {
		clone.Search.DefaultMetadataFilters[key] = append([]string(nil), values...)
	}
	return clone
}

func cloneCommandTemplate(src config.CommandTemplate) config.CommandTemplate {
	clone := config.CommandTemplate{
		Exec: src.Exec,
		Args: append([]string(nil), src.Args...),
	}
	if src.Wait != nil {
		wait := *src.Wait
		clone.Wait = &wait
	}
	if src.Silence != nil {
		silence := *src.Silence
		clone.Silence = &silence
	}
	return clone
}

func cloneHookCommands(src []config.CommandTemplate) []config.CommandTemplate {
	if len(src) == 0 {
		return nil
	}
	out := make([]config.CommandTemplate, 0, len(src))
	for _, template := range src {
		out = append(out, cloneCommandTemplate(template))
	}
	return out
}
