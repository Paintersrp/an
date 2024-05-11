package root

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/cmd/addSubdir"
	"github.com/Paintersrp/an/pkg/cmd/archive"
	"github.com/Paintersrp/an/pkg/cmd/echo"
	"github.com/Paintersrp/an/pkg/cmd/initialize"
	"github.com/Paintersrp/an/pkg/cmd/journal"
	"github.com/Paintersrp/an/pkg/cmd/new"
	"github.com/Paintersrp/an/pkg/cmd/notes"
	"github.com/Paintersrp/an/pkg/cmd/open"
	"github.com/Paintersrp/an/pkg/cmd/pin"
	"github.com/Paintersrp/an/pkg/cmd/settings"
	"github.com/Paintersrp/an/pkg/cmd/symlink"
	"github.com/Paintersrp/an/pkg/cmd/tags"
	"github.com/Paintersrp/an/pkg/cmd/tasks"
	"github.com/Paintersrp/an/pkg/cmd/todo"
	"github.com/Paintersrp/an/pkg/cmd/trash"
	"github.com/Paintersrp/an/pkg/cmd/unarchive"
	"github.com/Paintersrp/an/pkg/cmd/untrash"
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
		// Run notes tui by default, or leave as help?
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

	// TODO: Subdirectory creation is being asked even on the init command, should prob find a way to avoid that
	handleSubdirs(s.Config)

	// Add Child Commands to Root
	cmd.AddCommand(
		initialize.NewCmdInit(s),
		addSubdir.NewCmdAddSubdir(s),
		new.NewCmdNew(s),
		open.NewCmdOpen(s.Config),
		tags.NewCmdTags(s.Config),
		tasks.NewCmdTasks(s),
		pin.NewCmdPin(s, "text"),
		echo.NewCmdEcho(s),
		settings.NewCmdSettings(s.Config),
		symlink.NewCmdSymlink(s),
		notes.NewCmdNotes(s),
		todo.NewCmdTodo(s.Config),
		archive.NewCmdArchive(s),
		unarchive.NewCmdUnarchive(s),
		trash.NewCmdTrash(s),
		untrash.NewCmdUntrash(s),
		journal.NewCmdJournal(s),
	)

	return cmd, nil
}

func handleSubdirs(c *config.Config) {
	mode := viper.GetString("fsmode")
	exists, err := verifySubdirExists()
	cobra.CheckErr(err)
	switch mode {
	case "strict":
		if !exists {
			fmt.Println("Error: Subdirectory", subdirName, "does not exist.")
			fmt.Println(
				"In strict mode, new subdirectories are included with the add-subdir command.",
			)
			os.Exit(1)
		}
	case "free":
		if !exists {
			c.AddSubdir(subdirName)
		}
	case "confirm":
		if !exists {
			getConfirmation(c)
		}
	default:
		if !exists {
			getConfirmation(c)
		}
	}
}

func verifySubdirExists() (bool, error) {
	var subdirs []string
	if err := viper.UnmarshalKey("subdirs", &subdirs); err != nil {
		return false, err
	}

	for _, subdir := range subdirs {
		if subdir == subdirName {
			return true, nil
		}
	}

	return false, nil
}

func getConfirmation(c *config.Config) {
	var response string
	for {
		fmt.Printf(
			"Subdirectory %s does not exist.\nDo you want to create it?\n(y/n): ",
			subdirName,
		)
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "yes", "y":
			c.AddSubdir(subdirName)
			return
		case "no", "n":
			fmt.Println("Exiting due to non-existing subdirectory")
			os.Exit(0)
		default:
			fmt.Println("Invalid response. Please enter 'y'/'yes' or 'n'/'no'.")
		}
	}
}
