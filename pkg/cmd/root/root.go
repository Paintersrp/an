package root

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/cmd/addSubdir"
	"github.com/Paintersrp/an/pkg/cmd/archive"
	"github.com/Paintersrp/an/pkg/cmd/capture"
	"github.com/Paintersrp/an/pkg/cmd/echo"
	"github.com/Paintersrp/an/pkg/cmd/initialize"
	"github.com/Paintersrp/an/pkg/cmd/journal"
	"github.com/Paintersrp/an/pkg/cmd/new"
	"github.com/Paintersrp/an/pkg/cmd/notes"
	"github.com/Paintersrp/an/pkg/cmd/open"
	"github.com/Paintersrp/an/pkg/cmd/pin"
        "github.com/Paintersrp/an/pkg/cmd/review"
        "github.com/Paintersrp/an/pkg/cmd/settings"
	"github.com/Paintersrp/an/pkg/cmd/symlink"
	"github.com/Paintersrp/an/pkg/cmd/tags"
	"github.com/Paintersrp/an/pkg/cmd/tasks"
	"github.com/Paintersrp/an/pkg/cmd/trash"
	"github.com/Paintersrp/an/pkg/cmd/unarchive"
	"github.com/Paintersrp/an/pkg/cmd/untrash"
	"github.com/Paintersrp/an/pkg/cmd/views"
	"github.com/Paintersrp/an/pkg/cmd/workspace"
)

var subdirName string

func NewCmdRoot(s *state.State) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:     "atomic",
		Aliases: []string{"an", "a-n"},
		Short:   "Launch into writing atomic notes, blended into integration with Obsidian.",
		Long: `A utility to help you get into the habit of writing notes by providing ways to quickly
  get up and writing with atomic notes. 

              [title]  [tags]
  an new robotics "robotics science class study-notes"
  `,
		RunE: notes.NewCmdNotes(s).RunE,
	}

	cmd.PersistentFlags().
		StringVarP(
			&subdirName,
			"subdir",
			"s",
			"atoms",
			"Subdirectory to use for this command.",
		)
	viper.BindPFlag("subdir", cmd.PersistentFlags().Lookup("subdir"))

	workspaceDefault := ""
	if s != nil {
		workspaceDefault = s.WorkspaceName
	}
	cmd.PersistentFlags().StringP(
		"workspace",
		"w",
		workspaceDefault,
		"Workspace to use for this invocation.",
	)
	viper.BindPFlag("workspace", cmd.PersistentFlags().Lookup("workspace"))

	cmd.AddCommand(
		initialize.NewCmdInit(s),
		addSubdir.NewCmdAddSubdir(s),
		new.NewCmdNew(s),
		open.NewCmdOpen(s.Config),
		tags.NewCmdTags(s.Config),
		tasks.NewCmdTasks(s),
		pin.NewCmdPin(s, "text"),
		echo.NewCmdEcho(s),
                capture.NewCmdCapture(s),
                review.NewCmdReview(s),
                settings.NewCmdSettings(s),
		symlink.NewCmdSymlink(s),
		notes.NewCmdNotes(s),
		archive.NewCmdArchive(s),
		unarchive.NewCmdUnarchive(s),
		trash.NewCmdTrash(s),
		untrash.NewCmdUntrash(s),
		journal.NewCmdJournal(s),
		views.NewCmdViews(s),
		workspace.NewCmdWorkspace(s),
	)

	return cmd, nil
}
