package taskPin

import (
	"github.com/spf13/cobra"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/cmd/pin"
)

func NewCmdTaskPin(c *config.Config) *cobra.Command {
	return pin.NewCmdPin(c, "task")
}
