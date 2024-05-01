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
	VaultDir       string   `json:"vault_dir"        yaml:"vaultdir"`
	Editor         string   `json:"editor"           yaml:"editor"`
	NvimArgs       string   `json:"nvim_args"        yaml:"nvimargs"`
	SubDirs        []string `json:"sub_dirs"         yaml:"subdirs"`
	FileSystemMode string   `json:"fs_mode"          yaml:"fsmode"`
	PinnedFile     string   `json:"pinned_file"      yaml:"pinned_file"`
	PinnedTaskFile string   `json:"pinned_task_file" yaml:"pinned_task_file"`
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
	return cfg, nil
}

func (cfg *Config) ToFile() error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	p := cfg.GetConfigPath()
	fmt.Println(p)
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

func (cfg *Config) ChangePin(file, pinType string) {
	// TODO: Validation

	switch pinType {
	case "task":
		cfg.PinnedTaskFile = file
	case "text":
		cfg.PinnedFile = file
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

	fmt.Printf(
		"Pinned File changed to '%s' and configuration saved successfully.\n",
		file,
	)
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
