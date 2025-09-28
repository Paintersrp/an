package state

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	watcher   *fsnotify.Watcher
	vault     string
	done      chan struct{}
	once      sync.Once
	mu        sync.Mutex
	pending   []tea.Msg
	heartbeat func() tea.Cmd
	interval  time.Duration
	onChange  func(string)
	onClose   func()
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
		if msg := w.dequeuePending(); msg != nil {
			return msg
		}

		hb, interval := w.heartbeatConfig()
		var ticker *time.Ticker
		var ticks <-chan time.Time
		if hb != nil && interval > 0 {
			ticker = time.NewTicker(interval)
			ticks = ticker.C
			defer ticker.Stop()
		}

		for {
			select {
			case <-w.done:
				return nil
			case <-ticks:
				if msg := w.invokeHeartbeat(hb); msg != nil {
					return msg
				}
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

				if w.onChange != nil {
					w.onChange(rel)
				}

				if msg := w.invokeHeartbeat(hb); msg != nil {
					w.enqueuePending(VaultNoteChangedMsg{Path: rel})
					return msg
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

func (w *VaultWatcher) heartbeatConfig() (func() tea.Cmd, time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.heartbeat, w.interval
}

func (w *VaultWatcher) invokeHeartbeat(fn func() tea.Cmd) tea.Msg {
	if fn == nil {
		return nil
	}
	cmd := fn()
	if cmd == nil {
		return nil
	}
	return cmd()
}

func (w *VaultWatcher) enqueuePending(msg tea.Msg) {
	w.mu.Lock()
	w.pending = append(w.pending, msg)
	w.mu.Unlock()
}

func (w *VaultWatcher) dequeuePending() tea.Msg {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.pending) == 0 {
		return nil
	}
	msg := w.pending[0]
	w.pending = w.pending[1:]
	return msg
}

func (w *VaultWatcher) Close() error {
	if w == nil {
		return nil
	}

	var closeErr error
	w.once.Do(func() {
		close(w.done)
		closeErr = w.watcher.Close()
		if w.onClose != nil {
			w.onClose()
		}
	})

	return closeErr
}

// OnChange registers a callback that receives relative note paths whenever the
// watcher detects a relevant change.
func (w *VaultWatcher) OnChange(fn func(string)) {
	if w == nil {
		return
	}
	w.onChange = fn
}

// OnClose registers a callback that is invoked exactly once when the watcher
// shuts down.
func (w *VaultWatcher) OnClose(fn func()) {
	if w == nil {
		return
	}
	w.onClose = fn
}

// SetHeartbeat configures a command that is invoked whenever the watcher
// detects a change event or when the periodic ticker fires.
func (w *VaultWatcher) SetHeartbeat(fn func() tea.Cmd, interval time.Duration) {
	if w == nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.heartbeat = fn
	w.interval = interval
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
