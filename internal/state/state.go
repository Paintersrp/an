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

type ViewShortcut struct {
	View  string
	Label string
}

type RootStatus struct {
	ActiveView    string
	WorkspaceName string
	WorkspaceHint string
	Shortcuts     []ViewShortcut
	Footer        string
}

type State struct {
	Config        *config.Config
	Workspace     *config.Workspace
	WorkspaceName string
	Templater     *templater.Templater
	Handler       *handler.FileHandler
	ViewManager   *views.ViewManager
	Views         map[string]views.View
	Home          string
	Vault         string
	Watcher       *VaultWatcher
	RootStatus    RootStatus
}

func NewState(workspaceOverride string) (*State, error) {
	home, err := GetHomeDir()
	if err != nil {
		return nil, err
	}

	cfg, err := LoadConfig(home)
	if err != nil {
		return nil, err
	}

	if workspaceOverride != "" {
		if err := cfg.ActivateWorkspace(workspaceOverride); err != nil {
			return nil, err
		}
	}

	ws, err := cfg.ActiveWorkspace()
	if err != nil {
		return nil, err
	}

	t, err := templater.NewTemplater(ws)
	if err != nil {
		return nil, fmt.Errorf("failed to create templater: %v", err)
	}

	h := handler.NewFileHandler(ws.VaultDir)
	vm, err := views.NewViewManager(h, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to configure views: %w", err)
	}

	watcher, err := NewVaultWatcher(ws.VaultDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault watcher: %w", err)
	}

	return &State{
		Config:        cfg,
		Workspace:     ws,
		WorkspaceName: cfg.CurrentWorkspace,
		Templater:     t,
		Handler:       h,
		ViewManager:   vm,
		Views:         vm.Views,
		Home:          home,
		Vault:         ws.VaultDir,
		Watcher:       watcher,
		RootStatus:    RootStatus{},
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
	// TODO: Eventually will factor out Viper entirely
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
