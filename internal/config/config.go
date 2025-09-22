package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Paintersrp/an/internal/pin"
	"github.com/spf13/cobra"
)

type PinMap map[string]string

type Config struct {
	PinManager     *pin.PinManager `yaml:"-"`
	NamedPins      PinMap          `yaml:"named_pins"       json:"named_pins"`
	NamedTaskPins  PinMap          `yaml:"named_task_pins"  json:"named_task_pins"`
	VaultDir       string          `yaml:"vaultdir"         json:"vault_dir"`
	Editor         string          `yaml:"editor"           json:"editor"`
	NvimArgs       string          `yaml:"nvimargs"         json:"nvim_args"`
	FileSystemMode string          `yaml:"fsmode"           json:"fs_mode"`
	PinnedFile     string          `yaml:"pinned_file"      json:"pinned_file"`
	PinnedTaskFile string          `yaml:"pinned_task_file" json:"pinned_task_file"`
	SubDirs        []string        `yaml:"subdirs"          json:"sub_dirs"`
}

var ValidModes = map[string]bool{
	"strict":  true,
	"confirm": true,
	"free":    true,
}

var validEditorNames = []string{"nvim", "obsidian", "vscode", "vim", "nano"}

var ValidEditors = func() map[string]bool {
	editors := make(map[string]bool, len(validEditorNames))
	for _, editor := range validEditorNames {
		editors[editor] = true
	}

	return editors
}()

func ValidateEditor(editor string) error {
	if _, valid := ValidEditors[editor]; valid {
		return nil
	}

	return fmt.Errorf(
		"invalid editor: %q. Please choose from %s.",
		editor,
		validEditorList(),
	)
}

func validEditorList() string {
	quoted := make([]string, len(validEditorNames))
	for i, name := range validEditorNames {
		quoted[i] = fmt.Sprintf("'%s'", name)
	}

	if len(quoted) == 0 {
		return ""
	}

	if len(quoted) == 1 {
		return quoted[0]
	}

	return strings.Join(quoted[:len(quoted)-1], ", ") + ", or " + quoted[len(quoted)-1]
}

func Load(home string) (*Config, error) {
	path := GetConfigPath(home)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Editor != "" {
		if err := ValidateEditor(cfg.Editor); err != nil {
			return nil, err
		}
	}

	if cfg.NamedPins == nil {
		cfg.NamedPins = make(PinMap)
	}
	if cfg.NamedTaskPins == nil {
		cfg.NamedTaskPins = make(PinMap)
	}

	cfg.PinManager = pin.NewPinManager(
		pin.PinMap(cfg.NamedPins),
		pin.PinMap(cfg.NamedTaskPins),
		cfg.PinnedFile,
		cfg.PinnedTaskFile,
	)

	return cfg, nil
}

func (cfg *Config) GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return GetConfigPath(homeDir)
}

func (cfg *Config) AddSubdir(name string) error {
	for _, subDir := range cfg.SubDirs {
		if subDir == name {
			return fmt.Errorf("subdirectory %q already exists", name)
		}
	}

	cfg.SubDirs = append(cfg.SubDirs, name)
	return cfg.Save()
}

func (cfg *Config) ChangeMode(mode string) error {
	if _, valid := ValidModes[mode]; !valid {
		return fmt.Errorf(
			"invalid mode: %q. Please choose from 'strict', 'confirm', or 'free'",
			mode,
		)
	}

	cfg.FileSystemMode = mode
	return cfg.Save()
}

func (cfg *Config) ChangeEditor(editor string) error {
	if err := ValidateEditor(editor); err != nil {
		return err
	}

	cfg.Editor = editor
	return cfg.Save()
}

func (cfg *Config) AddPin(pinName, file, pinType string) error {
	err := cfg.PinManager.AddPin(pinName, file, pinType)
	if err != nil {
		return err
	}

	return cfg.syncPinsAndSave()
}

func (cfg *Config) ChangePin(file, pinType, pinName string) error {
	err := cfg.PinManager.ChangePin(file, pinType, pinName)
	if err != nil {
		return err
	}

	return cfg.syncPinsAndSave()
}

func (cfg *Config) DeleteNamedPin(pinName, pinType string) error {
	err := cfg.PinManager.DeleteNamedPin(pinName, pinType)
	if err != nil {
		return err
	}

	return cfg.syncPinsAndSave()
}

func (cfg *Config) ClearPinnedFile(pinType string) error {
	err := cfg.PinManager.ClearPinnedFile(pinType)
	if err != nil {
		return err
	}

	return cfg.syncPinsAndSave()
}

func (cfg *Config) RenamePin(oldName, newName, pinType string) error {
	err := cfg.PinManager.RenamePin(oldName, newName, pinType)
	if err != nil {
		return err
	}

	return cfg.syncPinsAndSave()
}

func (cfg *Config) ListPins(pinType string) error {
	err := cfg.PinManager.ListPins(pinType)
	if err != nil {
		return err
	}

	return nil
}

func (cfg *Config) Save() error {
	if cfg.Editor != "" {
		if err := ValidateEditor(cfg.Editor); err != nil {
			return err
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	configPath := cfg.GetConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o644)
}

func (cfg *Config) syncPins() {
	cfg.NamedPins = PinMap(cfg.PinManager.NamedPins)
	cfg.NamedTaskPins = PinMap(cfg.PinManager.NamedTaskPins)
	cfg.PinnedFile = cfg.PinManager.PinnedFile
	cfg.PinnedTaskFile = cfg.PinManager.PinnedTaskFile
}

func (cfg *Config) syncPinsAndSave() error {
	cfg.syncPins()
	return cfg.Save()
}

func (cfg *Config) HandleSubdir(subdirName string) {
	exists, err := verifySubdirExists(subdirName)
	cobra.CheckErr(err)

	switch cfg.FileSystemMode {
	case "strict":
		if !exists {
			fmt.Println("Error: Subdirectory", subdirName, "does not exist.")
			fmt.Println(
				"In strict mode, new subdirectories are included with the add-subdir command.",
			)
			os.Exit(1)
		}
	case "free":
		if !exists {
			cfg.AddSubdir(subdirName)
		}
	case "confirm":
		if !exists {
			cfg.getConfirmation(subdirName)
		}
	default:
		if !exists {
			cfg.getConfirmation(subdirName)
		}
	}
}

func (cfg *Config) getConfirmation(subdirName string) {
	var response string
	for {
		fmt.Printf(
			"Subdirectory %s does not exist.\nDo you want to create it?\n(y/n): ",
			subdirName,
		)
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "yes", "y":
			cfg.AddSubdir(subdirName)
			return
		case "no", "n":
			fmt.Println("Exiting due to non-existing subdirectory")
			os.Exit(0)
		default:
			fmt.Println("Invalid response. Please enter 'y'/'yes' or 'n'/'no'.")
		}
	}
}
