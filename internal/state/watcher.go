package state

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"

	"github.com/Paintersrp/an/internal/pathutil"
)

type VaultNoteChangedMsg struct {
	Path string
}

type VaultWatcherErrMsg struct {
	Err error
}

type VaultWatcher struct {
	watcher *fsnotify.Watcher
	vault   string
	done    chan struct{}
	once    sync.Once
}

func NewVaultWatcher(vault string) (*VaultWatcher, error) {
	normalizedVault := pathutil.NormalizePath(vault)
	if normalizedVault == "" {
		return nil, errors.New("vault directory cannot be empty")
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &VaultWatcher{
		watcher: w,
		vault:   normalizedVault,
		done:    make(chan struct{}),
	}

	if err := watcher.addRecursive(normalizedVault); err != nil {
		_ = watcher.Close()
		return nil, err
	}

	return watcher, nil
}

func (w *VaultWatcher) Start() tea.Cmd {
	if w == nil {
		return nil
	}

	return func() tea.Msg {
		for {
			select {
			case <-w.done:
				return nil
			case event, ok := <-w.watcher.Events:
				if !ok {
					return nil
				}

				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						_ = w.addRecursive(event.Name)
						continue
					}
				}

				if !w.isRelevant(event) {
					continue
				}

				rel, err := w.relativePath(event.Name)
				if err != nil || rel == "" {
					continue
				}

				return VaultNoteChangedMsg{Path: rel}
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return nil
				}
				if err != nil {
					return VaultWatcherErrMsg{Err: err}
				}
			}
		}
	}
}

func (w *VaultWatcher) Close() error {
	if w == nil {
		return nil
	}

	var closeErr error
	w.once.Do(func() {
		close(w.done)
		closeErr = w.watcher.Close()
	})

	return closeErr
}

func (w *VaultWatcher) addRecursive(root string) error {
	normalized := pathutil.NormalizePath(root)
	return filepath.WalkDir(normalized, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return filepath.SkipDir
			}
			return err
		}

		if !d.IsDir() {
			return nil
		}

		return w.watcher.Add(path)
	})
}

func (w *VaultWatcher) isRelevant(event fsnotify.Event) bool {
	if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
		return false
	}

	rel, err := w.relativePath(event.Name)
	if err != nil || rel == "" {
		return false
	}

	return strings.EqualFold(filepath.Ext(rel), ".md")
}

func (w *VaultWatcher) relativePath(path string) (string, error) {
	normalized := pathutil.NormalizePath(path)
	rel, err := pathutil.VaultRelative(w.vault, normalized)
	if err != nil {
		return "", err
	}

	if rel == "." || rel == "" || strings.HasPrefix(rel, "..") {
		return "", nil
	}

	return rel, nil
}
