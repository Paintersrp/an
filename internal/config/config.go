package config

import (
	"errors"
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VaultDir string
	Editor   string
	NvimArgs string
	HomeDir  string
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

func ToFile(file_path string, cfg *Config) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	d := path.Dir(file_path)
	if _, err := os.Stat(d); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	if err := os.WriteFile(file_path, b, 0644); err != nil {
		return err
	}

	return nil
}
