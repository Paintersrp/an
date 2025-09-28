package state

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/constants"
	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/search"
	indexsvc "github.com/Paintersrp/an/internal/services/index"
	"github.com/Paintersrp/an/internal/templater"
	"github.com/Paintersrp/an/internal/views"
)

type State struct {
	Config        *config.Config
	Workspace     *config.Workspace
	WorkspaceName string
	Review        config.ReviewConfig
	Templater     *templater.Templater
	Handler       *handler.FileHandler
	ViewManager   *views.ViewManager
	Views         map[string]views.View
	Home          string
	Vault         string
	Watcher       *VaultWatcher
	Index         IndexService
	RootStatus    *RootStatus
}

type RootStatus struct {
	Line string
}

// IndexService exposes the shared search index snapshots produced by the
// workspace index manager.
type IndexService interface {
	AcquireSnapshot() (*search.Index, error)
	QueueUpdate(string)
	Stats() indexsvc.Stats
	Close() error
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

	searchCfg := search.Config{
		EnableBody:     ws.Search.EnableBody,
		IgnoredFolders: append([]string(nil), ws.Search.IgnoredFolders...),
	}
	indexService := indexsvc.NewService(ws.VaultDir, searchCfg)
	watcher.OnChange(func(rel string) {
		if indexService != nil {
			indexService.QueueUpdate(rel)
		}
	})
	watcher.OnClose(func() {
		if indexService != nil {
			_ = indexService.Close()
		}
	})

	return &State{
		Config:        cfg,
		Workspace:     ws,
		WorkspaceName: cfg.CurrentWorkspace,
		Review:        ws.Review,
		Templater:     t,
		Handler:       h,
		ViewManager:   vm,
		Views:         vm.Views,
		Home:          home,
		Vault:         ws.VaultDir,
		Watcher:       watcher,
		Index:         indexService,
		RootStatus:    &RootStatus{},
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

// Close releases resources associated with the state, including the vault
// watcher and shared index service.
func (s *State) Close() error {
	if s == nil {
		return nil
	}

	var errs []error
	if s.Watcher != nil {
		if err := s.Watcher.Close(); err != nil {
			errs = append(errs, err)
		}
		s.Watcher = nil
	}
	if s.Index != nil {
		if err := s.Index.Close(); err != nil && !errors.Is(err, indexsvc.ErrClosed) {
			errs = append(errs, err)
		}
		s.Index = nil
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
