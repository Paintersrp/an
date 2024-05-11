package config

import (
	"errors"
	"fmt"
	"os"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/Paintersrp/an/internal/constants"
)

type Config struct {
	NamedPins      map[string]string `json:"named_pins"       yaml:"named_pins"`
	NamedTaskPins  map[string]string `json:"named_task_pins"  yaml:"named_task_pins"`
	VaultDir       string            `json:"vault_dir"        yaml:"vaultdir"`
	Editor         string            `json:"editor"           yaml:"editor"`
	NvimArgs       string            `json:"nvim_args"        yaml:"nvimargs"`
	FileSystemMode string            `json:"fs_mode"          yaml:"fsmode"`
	PinnedFile     string            `json:"pinned_file"      yaml:"pinned_file"`
	PinnedTaskFile string            `json:"pinned_task_file" yaml:"pinned_task_file"`
	SubDirs        []string          `json:"sub_dirs"         yaml:"subdirs"`
}

var ValidModes = map[string]bool{
	"strict":  true,
	"confirm": true,
	"free":    true,
}

var ValidEditors = map[string]bool{
	"nvim": true, // The one true god
}

// We are using viper to laod the config, may not need this or the ToFile?
func FromFile(path string) (*Config, error) {
	cfg_file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(cfg_file, cfg); err != nil {
		return nil, err
	}

	if cfg.NamedPins == nil {
		cfg.NamedPins = make(map[string]string)
	}
	if cfg.NamedTaskPins == nil {
		cfg.NamedTaskPins = make(map[string]string)
	}

	return cfg, nil
}

func (cfg *Config) ToFile() error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	configPath := cfg.GetConfigPath()
	dir := path.Dir(configPath)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(configPath, data, 0644)
}

func (cfg *Config) AddSubdir(name string) {
	for _, subDir := range cfg.SubDirs {
		if subDir == name {
			fmt.Println("Subdirectory", name, "already exists.")
			return
		}
	}

	cfg.SubDirs = append(cfg.SubDirs, name)
	if err := cfg.ToFile(); err != nil {
		fmt.Println("Error saving the configuration:", err)
		return
	}

	fmt.Println("Subdirectory", name, "added successfully.")
}

func (cfg *Config) GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Home directory not found")
		os.Exit(1)
	}
	return StaticGetConfigPath(homeDir)
}

func (cfg *Config) ChangeMode(mode string) {
	if _, valid := ValidModes[mode]; !valid {
		fmt.Printf(
			"Invalid mode: %s. Please choose from 'strict', 'confirm', or 'free'.\n",
			mode,
		)
		return
	}

	cfg.FileSystemMode = mode
	if err := cfg.ToFile(); err != nil {
		fmt.Println("Error saving the configuration:", err)
		return
	}

	fmt.Printf("Mode changed to '%s' and configuration saved successfully.\n", mode)
}

func (cfg *Config) ChangeEditor(editor string) {
	if _, valid := ValidEditors[editor]; !valid {
		fmt.Printf("Invalid editor: %s. The only valid option is 'nvim'.\n", editor)
		return
	}

	cfg.Editor = editor
	if err := cfg.ToFile(); err != nil {
		fmt.Println("Error saving the configuration:", err)
		return
	}

	fmt.Printf("Editor changed to '%s' and configuration saved successfully.\n", editor)
}

func (cfg *Config) ChangePin(file, pinType, pinName string) {
	switch pinType {
	case "task":
		if pinName == "" {
			cfg.PinnedTaskFile = file
		} else {
			cfg.NamedTaskPins[pinName] = file
		}
	case "text":
		if pinName == "" {
			cfg.PinnedFile = file
		} else {
			cfg.NamedPins[pinName] = file
		}
	default:
		fmt.Println("Invalid Pin File Type. Valid options are text and task.")
		return
	}

	if err := cfg.ToFile(); err != nil {
		fmt.Println("Error saving the configuration:", err)
		return
	}

	if pinName != "" {
		fmt.Printf(
			"Name Pinned File '%s' changed to '%s' and configuration saved successfully.\n",
			pinName,
			file,
		)
	} else {
		fmt.Printf("Pinned File changed to '%s' and configuration saved successfully.\n", file)
	}
}

func (cfg *Config) DeleteNamedPin(pinName, pinType string, verbose bool) error {
	var pinMap map[string]string
	var message string

	switch pinType {
	case "task":
		pinMap = cfg.NamedTaskPins
		message = "Task pin '%s' deleted successfully."
	case "text":
		pinMap = cfg.NamedPins
		message = "Text pin '%s' deleted successfully."
	default:
		return fmt.Errorf(
			"invalid pin type: %s. Valid options are 'text' and 'task'",
			pinType,
		)
	}

	if _, exists := pinMap[pinName]; !exists {
		return fmt.Errorf("%s pin '%s' does not exist", pinType, pinName)
	}

	delete(pinMap, pinName)
	if err := cfg.ToFile(); err != nil {
		return fmt.Errorf("error saving the configuration: %s", err)
	}

	if verbose {
		fmt.Printf(message+"\n", pinName)
	}

	return nil
}

func (cfg *Config) ClearPinnedFile(pinType string, verbose bool) error {
	switch pinType {
	case "task":
		cfg.PinnedTaskFile = ""
		if verbose {
			fmt.Println("Pinned task file cleared successfully.")
		}
	case "text":
		cfg.PinnedFile = ""
		if verbose {
			fmt.Println("Pinned text file cleared successfully.")
		}
	default:
		return fmt.Errorf(
			"invalid pin type: %s. Valid options are 'text' and 'task'",
			pinType,
		)
	}

	if err := cfg.ToFile(); err != nil {
		return fmt.Errorf("error saving the configuration: %s", err)
	}

	return nil
}

func (cfg *Config) RenamePin(oldName, newName, pinType string, verbose bool) error {
	if oldName == "" || newName == "" {
		return fmt.Errorf("old name and new name must be provided")
	}
	if oldName == newName {
		return fmt.Errorf("new name is the same as old name")
	}

	var pinMap map[string]string

	switch pinType {
	case "task":
		pinMap = cfg.NamedTaskPins
	case "text":
		pinMap = cfg.NamedPins
	default:
		return fmt.Errorf(
			"invalid pin type: %s. Valid options are 'text' and 'task'",
			pinType,
		)
	}

	if _, exists := pinMap[oldName]; !exists {
		return fmt.Errorf("%s pin '%s' does not exist", pinType, oldName)
	}

	pinMap[newName] = pinMap[oldName]
	delete(pinMap, oldName)

	if err := cfg.ToFile(); err != nil {
		return fmt.Errorf("error saving the configuration: %s", err)
	}

	if verbose {
		fmt.Printf(
			"%s pin '%s' renamed to '%s' and configuration saved successfully.\n",
			pinType,
			oldName,
			newName,
		)
	}

	return nil
}

func (cfg *Config) ListPins(pinType string) error {
	var pins map[string]string
	var defaultPin string

	switch pinType {
	case "task":
		pins = cfg.NamedTaskPins
		defaultPin = cfg.PinnedTaskFile
	case "text":
		pins = cfg.NamedPins
		defaultPin = cfg.PinnedFile
	default:
		return fmt.Errorf(
			"invalid pin type: %s. Valid options are 'text' and 'task'",
			pinType,
		)
	}

	if defaultPin != "" {
		fmt.Printf("  Default: \n    - %s\n  Named:\n", defaultPin)
	}

	if len(pins) == 0 {
		fmt.Println("  No named pins available.")
		return nil
	}

	for name, file := range pins {
		fmt.Printf("    - %s: %s\n", name, file)
	}

	return nil
}

func StaticGetConfigPath(homeDir string) string {
	return fmt.Sprintf(
		"%s%s%s.%s",
		homeDir,
		constants.ConfigDir,
		constants.ConfigFile,
		constants.ConfigFileType,
	)
}

func EnsureConfigExists(home string) {
	dir := fmt.Sprintf("%s/%s", home, constants.ConfigDir)
	filePath := fmt.Sprintf(
		"%s/%s.%s",
		dir,
		constants.ConfigFile,
		constants.ConfigFileType,
	)

	if _, dirErr := os.Stat(dir); os.IsNotExist(dirErr) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			fmt.Printf("failed to create config directory.\nerror: %s", err)
			os.Exit(1)
		}
	}

	if _, fileErr := os.Stat(filePath); os.IsNotExist(fileErr) {
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("Error: failed to create config file. \nerror: %s", err)
			os.Exit(1)
		}
		file.Close()
	}
}
