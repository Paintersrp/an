package state

import (
	"fmt"
	"os"

	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/constants"
	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/templater"
	"github.com/Paintersrp/an/internal/views"
)

type State struct {
	Config      *config.Config
	Templater   *templater.Templater
	Handler     *handler.FileHandler
	ViewManager *views.ViewManager
	Views       map[string]views.View
	Home        string
	Vault       string
}

func NewState() (*State, error) {
	home, err := GetHomeDir()

	if err != nil {
		return nil, err
	}

	cfg, err := LoadConfig(home)

	if err != nil {
		return nil, err
	}

	t, err := templater.NewTemplater()
	if err != nil {
		return nil, fmt.Errorf("failed to create templater: %v", err)
	}

	h := handler.NewFileHandler(cfg.VaultDir)
	vm := views.NewViewManager(h, cfg.VaultDir)

	return &State{
		Config:      cfg,
		Templater:   t,
		Handler:     h,
		ViewManager: vm,
		Views:       vm.Views,
		Home:        home,
		Vault:       cfg.VaultDir,
	}, nil
}

func GetHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory. err: %s", err)
	}

	return home, nil
}

func LoadConfig(home string) (*config.Config, error) {
	// Eventually will factor out Viper entirely
	viper.AddConfigPath(home + constants.ConfigDir)
	viper.SetConfigName(constants.ConfigFile)
	viper.SetConfigType(constants.ConfigFileType)
	viper.ReadInConfig()

	err := config.EnsureConfigExists(home)
	if err != nil {
		return nil, err
	}

	return config.Load(home)
}
