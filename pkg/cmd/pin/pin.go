package pin

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/cmd/pin/pinAdd"
	"github.com/Paintersrp/an/pkg/cmd/pin/pinList"
	"github.com/Paintersrp/an/pkg/cmd/pin/pinRemove"
	"github.com/Paintersrp/an/pkg/cmd/pin/pinTable"
	"github.com/spf13/cobra"
)

func NewCmdPin(c *config.Config, pinType string) *cobra.Command {
	pinCommand := ""

	switch pinType {
	case "task":
		pinCommand = "tasks pin"
	case "text":
		pinCommand = "pin"
	}

	cmd := &cobra.Command{
		Use:     "pin",
		Aliases: []string{"p"},
		Short:   "Manage your pinned items.",
		Long: heredoc.Doc(fmt.Sprintf(`
			The pin command group provides a set of subcommands to manage pins, which are references to files or tasks that you can quickly access and manipulate within the application. You can add, list, and remove pins using the respective subcommands.

			Examples:
			  an %s add --name my-note
			  an %s list
			  an %s remove --name my-note
		`, pinCommand, pinCommand, pinCommand)),
	}

	cmd.AddCommand(pinAdd.Command(c, pinType))
	cmd.AddCommand(pinRemove.Command(c, pinType))
	cmd.AddCommand(pinList.Command(c, pinType))
	cmd.AddCommand(pinTable.Command(c, pinType))

	return cmd
}
