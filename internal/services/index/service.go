package index

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Paintersrp/an/internal/pathutil"
	"github.com/Paintersrp/an/internal/search"
)

// ErrClosed signals that the index service has been shut down and cannot be
// used to produce new snapshots.
var ErrClosed = errors.New("index service closed")

// ErrUnavailable indicates that the search index has not been built yet.
var ErrUnavailable = errors.New("search index unavailable")

// Stats captures lightweight instrumentation about the shared index.
type Stats struct {
	LastRebuild time.Time
	Pending     int
}

// Service owns a shared search index for a workspace and coordinates
// incremental updates coming from the vault watcher.
type Service struct {
	mu          sync.RWMutex
	vault       string
	config      search.Config
	index       *search.Index
	pending     map[string]struct{}
	lastRebuild time.Time
	closed      bool

	now    func() time.Time
	stat   func(string) (fs.FileInfo, error)
	maxAge time.Duration
}

// NewService constructs a workspace-scoped index service rooted at the vault.
func NewService(vault string, cfg search.Config) *Service {
	normalized := pathutil.NormalizePath(vault)
	return &Service{
		vault:   normalized,
		config:  cfg,
		pending: make(map[string]struct{}),
		now:     time.Now,
		stat:    os.Stat,
		maxAge:  time.Hour,
	}
}

// AcquireSnapshot returns a thread-safe snapshot of the search index. The
// method rebuilds the index or applies pending updates as needed before cloning
// the in-memory representation.
func (s *Service) AcquireSnapshot() (*search.Index, error) {
	if s == nil {
		return nil, ErrUnavailable
	}

	if err := s.ensureFresh(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, ErrClosed
	}
	if s.index == nil {
		return nil, ErrUnavailable
	}

	return s.index.Clone(), nil
}

// QueueUpdate schedules a relative path for incremental reindexing.
func (s *Service) QueueUpdate(rel string) {
	if s == nil {
		return
	}

	trimmed := strings.TrimSpace(rel)
	if trimmed == "" {
		return
	}

	normalized := filepath.ToSlash(trimmed)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	if s.pending == nil {
		s.pending = make(map[string]struct{})
	}
	s.pending[normalized] = struct{}{}
}

// Stats returns instrumentation about the index lifecycle.
func (s *Service) Stats() Stats {
	if s == nil {
		return Stats{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return Stats{LastRebuild: s.lastRebuild, Pending: len(s.pending)}
}

// Close releases the service. Subsequent calls to AcquireSnapshot will return
// ErrClosed.
func (s *Service) Close() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true
	s.index = nil
	s.pending = nil
	return nil
}

func (s *Service) ensureFresh() error {
	if s == nil {
		return ErrUnavailable
	}

	s.mu.RLock()
	closed := s.closed
	needsRebuild := s.index == nil
	if !needsRebuild && s.maxAge > 0 {
		needsRebuild = s.now().Sub(s.lastRebuild) > s.maxAge
	}
	hasPending := len(s.pending) > 0
	s.mu.RUnlock()

	if closed {
		return ErrClosed
	}

	if needsRebuild {
		if err := s.rebuild(); err != nil {
			return err
		}
	}

	if hasPending {
		if err := s.applyPending(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) rebuild() error {
	paths, err := s.collectNotePaths()
	if err != nil {
		return err
	}

	idx := search.NewIndex(s.vault, s.config)
	if err := idx.Build(paths); err != nil {
		return fmt.Errorf("build search index: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrClosed
	}

	s.index = idx
	s.lastRebuild = s.now()
	return nil
}

func (s *Service) applyPending() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrClosed
	}
	if s.index == nil {
		return ErrUnavailable
	}
	if len(s.pending) == 0 {
		return nil
	}

	idx := s.index
	pending := s.pending
	s.pending = make(map[string]struct{})

	for rel := range pending {
		abs := filepath.Join(s.vault, filepath.FromSlash(rel))
		normalized := pathutil.NormalizePath(abs)
		if normalized == "" {
			continue
		}

		info, err := s.stat(normalized)
		switch {
		case err == nil:
			if info.IsDir() {
				if err := idx.Remove(normalized); err != nil {
					return fmt.Errorf("remove directory %s: %w", normalized, err)
				}
				continue
			}
			if err := idx.Update(normalized); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					if remErr := idx.Remove(normalized); remErr != nil {
						return fmt.Errorf("remove missing %s: %w", normalized, remErr)
					}
					continue
				}
				return fmt.Errorf("update %s: %w", normalized, err)
			}
		case errors.Is(err, fs.ErrNotExist):
			if err := idx.Remove(normalized); err != nil {
				return fmt.Errorf("remove missing %s: %w", normalized, err)
			}
		default:
			return fmt.Errorf("stat %s: %w", normalized, err)
		}
	}

	return nil
}

func (s *Service) collectNotePaths() ([]string, error) {
	if s.vault == "" {
		return nil, errors.New("vault directory cannot be empty")
	}

	ignored := make(map[string]struct{}, len(s.config.IgnoredFolders))
	for _, dir := range s.config.IgnoredFolders {
		ignored[strings.ToLower(dir)] = struct{}{}
	}

	paths := make([]string, 0)
	err := filepath.WalkDir(s.vault, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if strings.HasPrefix(name, ".") && path != s.vault {
				return filepath.SkipDir
			}
			if _, skip := ignored[name]; skip {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(paths)
	return paths, nil
}
