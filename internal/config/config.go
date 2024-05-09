package config

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/Paintersrp/an/internal/constants"
	"gopkg.in/yaml.v3"
)

type Config struct {
	VaultDir       string            `json:"vault_dir"        yaml:"vaultdir"`
	Editor         string            `json:"editor"           yaml:"editor"`
	NvimArgs       string            `json:"nvim_args"        yaml:"nvimargs"`
	SubDirs        []string          `json:"sub_dirs"         yaml:"subdirs"`
	FileSystemMode string            `json:"fs_mode"          yaml:"fsmode"`
	PinnedFile     string            `json:"pinned_file"      yaml:"pinned_file"`
	NamedPins      map[string]string `json:"named_pins"       yaml:"named_pins"`
	PinnedTaskFile string            `json:"pinned_task_file" yaml:"pinned_task_file"`
	NamedTaskPins  map[string]string `json:"named_task_pins"  yaml:"named_task_pins"`
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
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	p := cfg.GetConfigPath()

	d := path.Dir(p)
	if _, err := os.Stat(d); errors.Is(
		err,
		os.ErrNotExist,
	) {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	if err := os.WriteFile(p, b, 0644); err != nil {
		return err
	}

	return nil
}

func (cfg *Config) AddSubdir(name string) {
	// Check if the subdirectory already exists
	for _, subDir := range cfg.SubDirs {
		if subDir == name {
			fmt.Println("Subdirectory", name, "already exists.")
			return
		}
	}

	// Append the new sub directory
	cfg.SubDirs = append(cfg.SubDirs, name)
	cfg.ToFile()

	fmt.Println("Subdirectory", name, "added successfully.")
}

func (cfg *Config) GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Home directory not found")
		os.Exit(1)
	}
	return fmt.Sprintf(
		"%s%s%s.%s",
		home,
		constants.ConfigDir,
		constants.ConfigFile,
		constants.ConfigFileType,
	)
}

func (cfg *Config) ChangeMode(mode string) {
	// Validate the input mode
	if _, valid := ValidModes[mode]; !valid {
		fmt.Printf(
			"Invalid mode: %s. Please choose from 'strict', 'confirm', or 'free'.\n",
			mode,
		)
		return
	}

	// Update the struct with the new mode
	cfg.FileSystemMode = mode

	// Save the updated configuration to file
	err := cfg.ToFile()
	if err != nil {
		fmt.Println("Error saving the configuration:", err)
		return
	}

	fmt.Printf(
		"Mode changed to '%s' and configuration saved successfully.\n",
		mode,
	)
}

func (cfg *Config) ChangeEditor(editor string) {
	// Validate the input editor
	if _, valid := ValidEditors[editor]; !valid {
		fmt.Printf(
			"Invalid editor: %s. The only valid option is 'nvim'.\n",
			editor,
		)
		return
	}

	// Update the struct with the new editor
	cfg.Editor = editor

	// Save the updated configuration to file
	err := cfg.ToFile()
	if err != nil {
		fmt.Println("Error saving the configuration:", err)
		return
	}

	fmt.Printf(
		"Editor changed to '%s' and configuration saved successfully.\n",
		editor,
	)
}

func (cfg *Config) ChangePin(file, pinType, pinName string) {
	// TODO: Validation

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

	// Save the updated configuration to file
	err := cfg.ToFile()
	if err != nil {
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
		fmt.Printf(
			"Pinned File changed to '%s' and configuration saved successfully.\n",
			file,
		)
	}
}

func (cfg *Config) DeleteNamedPin(pinName string, pinType string, verbose bool) error {
	switch pinType {
	case "task":
		if _, exists := cfg.NamedTaskPins[pinName]; exists {
			delete(cfg.NamedTaskPins, pinName)
			if verbose {
				fmt.Printf("Task pin '%s' deleted successfully.\n", pinName)
			}
		} else {
			return fmt.Errorf("task pin '%s' does not exist", pinName)
		}
	case "text":
		if _, exists := cfg.NamedPins[pinName]; exists {
			delete(cfg.NamedPins, pinName)
			if verbose {
				fmt.Printf("Text pin '%s' deleted successfully.\n", pinName)
			}
		} else {
			return fmt.Errorf("text pin '%s' does not exist", pinName)
		}
	default:
		return fmt.Errorf(
			"invalid pin type: %s. Valid options are 'text' and 'task'",
			pinType,
		)
	}

	// Save the updated configuration to file
	if err := cfg.ToFile(); err != nil {
		return fmt.Errorf("error saving the configuration: %s", err)
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

	// Save the updated configuration to file
	if err := cfg.ToFile(); err != nil {
		return fmt.Errorf("error saving the configuration: %s", err)
	}

	return nil
}

func (cfg *Config) RenamePin(
	oldName, newName, pinType string,
	verbose bool,
) error {
	// Validate input
	if oldName == "" || newName == "" {
		return fmt.Errorf("old name and new name must be provided")
	}
	if oldName == newName {
		return fmt.Errorf("new name is the same as old name")
	}

	var pinMap map[string]string

	// Select the appropriate pin map based on pin type
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

	// Check if the old pin name exists
	if _, exists := pinMap[oldName]; !exists {
		return fmt.Errorf("%s pin '%s' does not exist", pinType, oldName)
	}

	// Rename the pin
	pinMap[newName] = pinMap[oldName]

	// Delete the old pin
	delete(pinMap, oldName)

	// Save the updated configuration to file
	if err := cfg.ToFile(); err != nil {
		return fmt.Errorf("error saving the configuration: %s", err)
	}

	if verbose {
		fmt.Printf("%s pin '%s' renamed to '%s'", pinType, oldName, newName)
		fmt.Println(" and configuration saved successfully.")
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
	// Get the directory path of the file and absolute file path
	dir := fmt.Sprintf("%s/%s", home, constants.ConfigDir)
	filePath := fmt.Sprintf(
		"%s/%s.%s",
		dir,
		constants.ConfigFile,
		constants.ConfigFileType,
	)

	// Check if the directory already exists
	_, dirErr := os.Stat(dir)
	if os.IsNotExist(dirErr) {
		// If the directory does not exist, create it
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			fmt.Printf("failed to create config directory.\nerror: %s", err)
			os.Exit(1)
		}
	}

	// Check if the file already exists
	_, fileErr := os.Stat(filePath)
	if os.IsNotExist(fileErr) {
		// If the file does not exist, create an empty file
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("Error: failed to create config file. \nerror: %s", err)
			os.Exit(1)
		}
		file.Close()
	}
}
