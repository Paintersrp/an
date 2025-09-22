package untrash

import (
	"io"
	"testing"

	"github.com/Paintersrp/an/internal/state"
)

func TestUntrashCommandRequiresArgument(t *testing.T) {
	s := &state.State{}
	cmd := NewCmdUntrash(s)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected an error when no path argument is provided")
	}
}
