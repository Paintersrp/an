package trash

import (
	"io"
	"testing"

	"github.com/Paintersrp/an/internal/state"
)

func TestTrashCommandRequiresArgument(t *testing.T) {
	s := &state.State{}
	cmd := NewCmdTrash(s)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected an error when no path argument is provided")
	}

	if err.Error() != "path argument is required" {
		t.Fatalf("expected error message %q, got %q", "path argument is required", err.Error())
	}
}
