package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/Paintersrp/an/internal/constants"
	"github.com/spf13/viper"
)

func GetConfigPath(homeDir string) string {
	return filepath.Join(
		homeDir,
		constants.ConfigDir,
		constants.ConfigFile+"."+constants.ConfigFileType,
	)
}

func EnsureConfigExists(homeDir string) error {
	configPath := GetConfigPath(homeDir)
	configDir := filepath.Dir(configPath)

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		file, err := os.Create(configPath)
		if err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
		file.Close()
	} else if err != nil {
		return fmt.Errorf("failed to check config file existence: %w", err)
	}

	cfg, err := Load(homeDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	requiredVars := []string{"VaultDir", "Editor", "FileSystemMode"}
	for _, varName := range requiredVars {
		value, err := getVarValue(cfg, varName)
		if err != nil {
			return err
		}
		if value == "" {
			return &ConfigInitError{
				msg: fmt.Sprintf("required config variable %q is not set", varName),
			}
		}
	}

	return nil
}

func getVarValue(cfg *Config, varName string) (string, error) {
	v := reflect.ValueOf(cfg).Elem()
	fieldVal := v.FieldByName(varName)

	if !fieldVal.IsValid() {
		return "", fmt.Errorf("config variable %q does not exist", varName)
	}

	if fieldVal.Kind() != reflect.String {
		return "", fmt.Errorf("config variable %q is not a string", varName)
	}

	return fieldVal.String(), nil
}

func verifySubdirExists(subdirName string) (bool, error) {
	var subdirs []string
	if err := viper.UnmarshalKey("subdirs", &subdirs); err != nil {
		return false, err
	}

	for _, subdir := range subdirs {
		if subdir == subdirName {
			return true, nil
		}
	}

	return false, nil
}
