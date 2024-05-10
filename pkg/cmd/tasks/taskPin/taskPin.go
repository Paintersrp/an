package taskPin

import (
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/pkg/cmd/pin"
)

func NewCmdTaskPin(s *state.State) *cobra.Command {
	return pin.NewCmdPin(s, "task")
}
