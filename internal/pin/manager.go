package pin

import (
	"errors"
	"fmt"
)

type PinMap map[string]string

type PinManager struct {
	NamedPins      PinMap
	NamedTaskPins  PinMap
	PinnedFile     string
	PinnedTaskFile string
}

func NewPinManager(
	namedPins, namedTaskPins PinMap,
	pinnedFile, pinnedTaskFile string,
) *PinManager {
	return &PinManager{
		NamedPins:      namedPins,
		NamedTaskPins:  namedTaskPins,
		PinnedFile:     pinnedFile,
		PinnedTaskFile: pinnedTaskFile,
	}
}

func (m *PinManager) ChangePin(file, pinType, pinName string) error {
	switch pinType {
	case "task":
		if pinName == "default" || pinName == "" {
			m.PinnedTaskFile = file
		} else {
			m.NamedTaskPins[pinName] = file
		}
	case "text":
		if pinName == "default" || pinName == "" {
			m.PinnedFile = file
		} else {
			m.NamedPins[pinName] = file
		}
	default:
		return fmt.Errorf("invalid pin file type. Valid options are text and task")
	}

	return nil
}

func (m *PinManager) DeleteNamedPin(pinName, pinType string) error {
	pinMap, err := m.getPinMap(pinType)
	if err != nil {
		return err
	}

	if _, exists := pinMap[pinName]; !exists {
		return fmt.Errorf("%s pin %q does not exist", pinType, pinName)
	}

	delete(pinMap, pinName)
	return nil
}

func (m *PinManager) ClearPinnedFile(pinType string) error {
	switch pinType {
	case "task":
		m.PinnedTaskFile = ""
	case "text":
		m.PinnedFile = ""
	default:
		return fmt.Errorf(
			"invalid pin type: %q. Valid options are 'text' and 'task'",
			pinType,
		)
	}
	return nil
}

func (m *PinManager) RenamePin(oldName, newName, pinType string) error {
	if oldName == "" || newName == "" {
		return errors.New("old name and new name must be provided")
	}
	if oldName == newName {
		return errors.New("new name is the same as old name")
	}

	pinMap, err := m.getPinMap(pinType)
	if err != nil {
		return err
	}

	if _, exists := pinMap[oldName]; !exists {
		return fmt.Errorf("%s pin %q does not exist", pinType, oldName)
	}

	pinMap[newName] = pinMap[oldName]
	delete(pinMap, oldName)

	return nil
}

func (m *PinManager) AddPin(pinName, file, pinType string) error {
	if pinName == "" {
		return errors.New("pin name must be provided")
	}
	if file == "" {
		return errors.New("file must be provided")
	}

	pinMap, err := m.getPinMap(pinType)
	if err != nil {
		return err
	}

	if _, exists := pinMap[pinName]; exists {
		return fmt.Errorf("%s pin %q already exists", pinType, pinName)
	}

	pinMap[pinName] = file
	return nil
}

func (m *PinManager) ListPins(pinType string) error {
	pinMap, defaultPin, err := m.getPinMapAndDefault(pinType)
	if err != nil {
		return err
	}

	if defaultPin != "" {
		fmt.Printf("  Default:\n    - %s\n", defaultPin)
	}

	if len(pinMap) == 0 {
		fmt.Println("  No named pins available.")
		return nil
	}

	fmt.Println("  Named:")
	for name, file := range pinMap {
		fmt.Printf("    - %s: %s\n", name, file)
	}
	return nil
}

func (m *PinManager) getPinMap(pinType string) (PinMap, error) {
	switch pinType {
	case "task":
		return m.NamedTaskPins, nil
	case "text":
		return m.NamedPins, nil
	default:
		return nil, fmt.Errorf(
			"invalid pin type: %q. Valid options are 'text' and 'task'",
			pinType,
		)
	}
}

func (m *PinManager) getPinMapAndDefault(pinType string) (PinMap, string, error) {
	pinMap, err := m.getPinMap(pinType)
	if err != nil {
		return nil, "", err
	}

	var defaultPin string
	switch pinType {
	case "task":
		defaultPin = m.PinnedTaskFile
	case "text":
		defaultPin = m.PinnedFile
	}

	return pinMap, defaultPin, nil
}
