package root

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/cmd/addMolecule"
	"github.com/Paintersrp/an/pkg/cmd/day"
	"github.com/Paintersrp/an/pkg/cmd/echo"
	"github.com/Paintersrp/an/pkg/cmd/initialize"
	"github.com/Paintersrp/an/pkg/cmd/new"
	"github.com/Paintersrp/an/pkg/cmd/open"
	"github.com/Paintersrp/an/pkg/cmd/openPin"
	"github.com/Paintersrp/an/pkg/cmd/pin"
	"github.com/Paintersrp/an/pkg/cmd/settings"
	"github.com/Paintersrp/an/pkg/cmd/tags"
	"github.com/Paintersrp/an/pkg/cmd/tasks"
	"github.com/Paintersrp/an/pkg/fs/templater"
)

var (
	moleculeName string
)

func NewCmdRoot(
	c *config.Config,
	t *templater.Templater,
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
	}

	// Validate the molecule flag
	cmd.PersistentFlags().
		StringVarP(
			&moleculeName,
			"molecule",
			"m",
			"atoms",
			"Molecule subdirectory to use for this command.",
		)

	viper.BindPFlag(
		"molecule",
		cmd.PersistentFlags().Lookup("molecule"),
	)

	// TODO: Molecule creation is being asked even on the init command, should prob find a way to avoid that
	handleMolecules(c)

	// Add Child Commands to Root
	cmd.AddCommand(initialize.NewCmdInit(c))
	cmd.AddCommand(addMolecule.NewCmdAddMolecule(c))
	cmd.AddCommand(new.NewCmdNew(c, t))
	cmd.AddCommand(open.NewCmdOpen(c))
	cmd.AddCommand(openPin.NewCmdOpenPin(c))
	cmd.AddCommand(tags.NewCmdTags(c))
	cmd.AddCommand(tasks.NewCmdTasks(c, t))
	cmd.AddCommand(day.NewCmdDay(c, t))
	cmd.AddCommand(pin.NewCmdPin(c))
	cmd.AddCommand(echo.NewCmdEcho(c))
	cmd.AddCommand(settings.NewCmdSettings(c))

	return cmd, nil
}

func handleMolecules(c *config.Config) {
	mode := viper.GetString("moleculeMode")
	exists, err := verifyMoleculeExists()
	cobra.CheckErr(err)
	switch mode {
	case "strict":
		if !exists {
			fmt.Println(
				"Error: Molecule",
				moleculeName,
				"does not exist.",
			)
			fmt.Println(
				"In strict mode, new molecules are added with the add-molecule command.",
			)
			os.Exit(1)
		}
	case "free":
		if !exists {
			c.AddMolecule(moleculeName)
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

func verifyMoleculeExists() (bool, error) {
	var molecules []string
	if err := viper.UnmarshalKey("molecules", &molecules); err != nil {
		fmt.Println("Error unmarshalling molecules:", err)
		return false, err
	}

	// Check if the specified molecule exists
	moleculeExists := false
	for _, molecule := range molecules {
		if molecule == moleculeName {
			moleculeExists = true
			break
		}
	}

	if !moleculeExists {
		return moleculeExists, nil
	}

	return moleculeExists, nil
}

func getConfirmation(c *config.Config) {
	var response string
	for {
		fmt.Printf(
			"Molecule %s does not exist.\nDo you want to create it?\n(y/n): ",
			moleculeName,
		)
		fmt.Scanln(&response)
		response = strings.ToLower(
			strings.TrimSpace(response),
		)

		switch response {
		case "yes", "y":
			// TODO: should use appCfg.AddMolecule
			c.AddMolecule(moleculeName)
			return
		case "no", "n":
			fmt.Println(
				"Exiting due to non-existing molecule",
			)
			os.Exit(0)
		default:
			fmt.Println(
				"Invalid response. Please enter 'y'/'yes' or 'n'/'no'.",
			)
		}
	}
}
