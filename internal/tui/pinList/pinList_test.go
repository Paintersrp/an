package pinList

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/pin"
	"github.com/Paintersrp/an/internal/state"
	"github.com/Paintersrp/an/internal/tui/notes"
	"github.com/Paintersrp/an/internal/tui/pinList/submodels/input"
	"github.com/Paintersrp/an/internal/tui/pinList/submodels/sublist"
)

func TestUpdateAddPinErrorShowsStatusAndKeepsDialog(t *testing.T) {
	cfg := &config.Config{
		NamedPins:     config.PinMap{"existing": "path"},
		NamedTaskPins: config.PinMap{},
	}
	cfg.PinManager = pin.NewPinManager(
		pin.PinMap(cfg.NamedPins),
		pin.PinMap(cfg.NamedTaskPins),
		"",
		"",
	)

	model := newTestPinListModel(cfg, "text")
	model.adding = true
	model.finding = true
	model.input.Input.SetValue("existing")
	model.sublist.List = list.New(
		[]list.Item{newNotesListItem("/tmp/path.md")},
		list.NewDefaultDelegate(),
		0,
		0,
	)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected status message command")
	}

	updatedModel, ok := updated.(PinListModel)
	if !ok {
		t.Fatalf("unexpected model type %T", updated)
	}

	if !updatedModel.adding {
		t.Fatalf("expected adding state to remain true")
	}

	if !updatedModel.finding {
		t.Fatalf("expected finding state to remain true")
	}

	status := readStatusMessage(updatedModel.list)
	if !strings.Contains(status, "Failed to add pin") {
		t.Fatalf("expected failure message, got %q", status)
	}
	if !strings.Contains(status, "already exists") {
		t.Fatalf("expected underlying error in message, got %q", status)
	}
}

func TestUpdateRenamePinErrorShowsStatusAndKeepsDialog(t *testing.T) {
	cfg := &config.Config{
		NamedPins:     config.PinMap{},
		NamedTaskPins: config.PinMap{},
	}
	cfg.PinManager = pin.NewPinManager(
		pin.PinMap(cfg.NamedPins),
		pin.PinMap(cfg.NamedTaskPins),
		"",
		"",
	)

	model := newTestPinListModel(cfg, "text")
	model.renaming = true
	model.renamingFor = "missing"
	model.input.Input.SetValue("new-name")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected status message command")
	}

	updatedModel, ok := updated.(PinListModel)
	if !ok {
		t.Fatalf("unexpected model type %T", updated)
	}

	if !updatedModel.renaming {
		t.Fatalf("expected renaming state to remain true")
	}

	status := readStatusMessage(updatedModel.list)
	if !strings.Contains(status, "Failed to rename pin") {
		t.Fatalf("expected failure message, got %q", status)
	}
	if !strings.Contains(status, "does not exist") {
		t.Fatalf("expected underlying error in message, got %q", status)
	}
}

func TestUpdateChangePinErrorShowsStatusAndKeepsDialog(t *testing.T) {
	cfg := &config.Config{
		NamedPins:     config.PinMap{},
		NamedTaskPins: config.PinMap{},
	}
	cfg.PinManager = pin.NewPinManager(
		pin.PinMap(cfg.NamedPins),
		pin.PinMap(cfg.NamedTaskPins),
		"",
		"",
	)

	model := newTestPinListModel(cfg, "invalid")
	model.finding = true
	model.findingFor = "default"
	model.sublist.List = list.New(
		[]list.Item{newNotesListItem("/tmp/path.md")},
		list.NewDefaultDelegate(),
		0,
		0,
	)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected status message command")
	}

	updatedModel, ok := updated.(PinListModel)
	if !ok {
		t.Fatalf("unexpected model type %T", updated)
	}

	if !updatedModel.finding {
		t.Fatalf("expected finding state to remain true")
	}

	status := readStatusMessage(updatedModel.list)
	if !strings.Contains(status, "Failed to change pin") {
		t.Fatalf("expected failure message, got %q", status)
	}
	if !strings.Contains(status, "invalid pin file type") {
		t.Fatalf("expected underlying error in message, got %q", status)
	}
}

func newTestPinListModel(cfg *config.Config, pinType string) PinListModel {
	state := &state.State{Config: cfg}
	return PinListModel{
		list:         list.New(nil, list.NewDefaultDelegate(), 0, 0),
		keys:         newListKeyMap(),
		delegateKeys: newDelegateKeyMap(),
		state:        state,
		pinType:      pinType,
		sublist:      sublist.SubListModel{List: list.New(nil, list.NewDefaultDelegate(), 0, 0)},
		input:        input.NewNameInput(),
	}
}

func newNotesListItem(path string) notes.ListItem {
	item := notes.ListItem{}
	setUnexportedString(&item, "path", path)
	setUnexportedString(&item, "fileName", "note.md")
	return item
}

func readStatusMessage(l list.Model) string {
	v := reflect.ValueOf(&l).Elem().FieldByName("statusMessage")
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Interface().(string)
}

func setUnexportedString(target interface{}, field, value string) {
	v := reflect.ValueOf(target).Elem().FieldByName(field)
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().SetString(value)
}
