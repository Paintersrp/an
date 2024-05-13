package pin

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/cmd/pin/pinAdd"
	"github.com/Paintersrp/an/pkg/cmd/pin/pinList"
	"github.com/Paintersrp/an/pkg/cmd/pin/pinOpen"
	"github.com/Paintersrp/an/pkg/cmd/pin/pinRemove"
)

func NewCmdPin(s *state.State, pinType string) *cobra.Command {
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

	cmd.AddCommand(pinAdd.NewCmdPinAdd(s, pinType))
	cmd.AddCommand(pinOpen.NewCmdPinOpen(s, pinType))
	cmd.AddCommand(pinRemove.NewCmdPinRemove(s, pinType))
	cmd.AddCommand(pinList.NewCmdPinList(s, pinType))

	return cmd
}
