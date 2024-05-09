package taskPin

import (
	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/pkg/cmd/pin"
	"github.com/spf13/cobra"
)

func NewCmdTaskPin(c *config.Config) *cobra.Command {
	return pin.NewCmdPin(c, "task")
}
