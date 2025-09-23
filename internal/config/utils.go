package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Paintersrp/an/internal/constants"
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

	if cfg.CurrentWorkspace == "" {
		return &ConfigInitError{msg: "no current workspace is configured"}
	}

	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return err
	}

	required := map[string]string{
		"VaultDir":       ws.VaultDir,
		"Editor":         ws.Editor,
		"FileSystemMode": ws.FileSystemMode,
	}

	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return &ConfigInitError{
				msg: fmt.Sprintf("required config variable %q is not set", name),
			}
		}
	}

	return nil
}
