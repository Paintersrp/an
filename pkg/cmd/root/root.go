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
	"github.com/Paintersrp/an/pkg/cmd/day"
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

var (
	subdirName string
)

func NewCmdRoot(
	s *state.State,
) (*cobra.Command, error) {
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

	// Validate the subdirectory flag
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
	cmd.AddCommand(initialize.NewCmdInit(s))
	cmd.AddCommand(addSubdir.NewCmdAddSubdir(s))
	cmd.AddCommand(new.NewCmdNew(s))
	cmd.AddCommand(open.NewCmdOpen(s.Config))
	cmd.AddCommand(tags.NewCmdTags(s.Config))
	cmd.AddCommand(tasks.NewCmdTasks(s))
	cmd.AddCommand(day.NewCmdDay(s))
	cmd.AddCommand(pin.NewCmdPin(s, "text"))
	cmd.AddCommand(echo.NewCmdEcho(s))
	cmd.AddCommand(settings.NewCmdSettings(s.Config))
	cmd.AddCommand(symlink.NewCmdSymlink(s.Config))
	cmd.AddCommand(notes.NewCmdNotes(s))
	cmd.AddCommand(todo.NewCmdTodo(s.Config))
	cmd.AddCommand(archive.NewCmdArchive(s))
	cmd.AddCommand(unarchive.NewCmdUnarchive(s))
	cmd.AddCommand(trash.NewCmdTrash(s))
	cmd.AddCommand(untrash.NewCmdUntrash(s))
	cmd.AddCommand(journal.NewCmdJournal(s))

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
		fmt.Println("Error unmarshalling subdirs:", err)
		return false, err
	}

	// Check if the specified subdirectory exists
	subdirExists := false
	for _, subdir := range subdirs {
		if subdir == subdirName {
			subdirExists = true
			break
		}
	}

	if !subdirExists {
		return subdirExists, nil
	}

	return subdirExists, nil
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
