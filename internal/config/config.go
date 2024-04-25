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
	VaultDir     string   `json:"vault_dir"     yaml:"vault_dir"`
	Editor       string   `json:"editor"        yaml:"editor"`
	NvimArgs     string   `json:"nvim_args"     yaml:"nvim_args"`
	HomeDir      string   `json:"home_dir"      yaml:"home_dir"`
	Molecules    []string `json:"molecules"     yaml:"molecules"`
	MoleculeMode string   `json:"molecule_mode" yaml:"molecule_mode"`
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

func (cfg *Config) AddMolecule(name string) {
	// Check if the molecule already exists
	for _, molecule := range cfg.Molecules {
		if molecule == name {
			fmt.Println("Molecule", name, "already exists.")
			return
		}
	}

	// Append the new molecule
	cfg.Molecules = append(cfg.Molecules, name)
	cfg.ToFile()

	fmt.Println("Molecule", name, "added successfully.")
}

func (cfg *Config) GetConfigPath() string {
	return fmt.Sprintf(
		"%s%s%s.%s",
		cfg.HomeDir,
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
	cfg.MoleculeMode = mode

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
