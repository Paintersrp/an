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

	"github.com/Paintersrp/an/internal/parser"
	"github.com/Paintersrp/an/internal/pathutil"
)

// ErrClosed signals that the task index service has been shut down.
var ErrClosed = errors.New("task index service closed")

// ErrUnavailable indicates that the task index has not been built yet.
var ErrUnavailable = errors.New("task index unavailable")

// Task represents a parsed markdown task tracked by the shared index.
type Task struct {
	Path     string
	Line     int
	Status   string
	Content  string
	Metadata parser.TaskMetadata
}

// Snapshot is an immutable view of the cached task index.
type Snapshot struct {
	tasks   map[string][]Task
	total   int
	created time.Time
}

// Tasks returns a flattened and sorted slice of tasks contained in the snapshot.
func (s *Snapshot) Tasks() []Task {
	if s == nil || len(s.tasks) == 0 {
		return nil
	}

	paths := make([]string, 0, len(s.tasks))
	for path := range s.tasks {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	tasks := make([]Task, 0, s.total)
	for _, path := range paths {
		entries := s.tasks[path]
		if len(entries) == 0 {
			continue
		}

		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Line == entries[j].Line {
				return entries[i].Content < entries[j].Content
			}
			return entries[i].Line < entries[j].Line
		})

		tasks = append(tasks, entries...)
	}

	return tasks
}

// Service owns the shared task index for a workspace.
type Service struct {
	mu      sync.RWMutex
	vault   string
	cache   map[string][]Task
	total   int
	pending map[string]struct{}
	closed  bool

	now           func() time.Time
	stat          func(string) (fs.FileInfo, error)
	parserFactory func(string) *parser.Parser

	maxTasks int
	maxNotes int
}

const (
	defaultMaxTasks = 100_000
	defaultMaxNotes = 10_000
)

// NewService constructs a workspace-scoped task index service rooted at the vault.
func NewService(vault string) *Service {
	normalized := pathutil.NormalizePath(vault)
	return &Service{
		vault:         normalized,
		pending:       make(map[string]struct{}),
		now:           time.Now,
		stat:          os.Stat,
		parserFactory: parser.NewParser,
		maxTasks:      defaultMaxTasks,
		maxNotes:      defaultMaxNotes,
	}
}

// AcquireSnapshot returns a thread-safe snapshot of the task index.
func (s *Service) AcquireSnapshot() (*Snapshot, error) {
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
	if s.cache == nil {
		return nil, ErrUnavailable
	}

	tasks := make(map[string][]Task, len(s.cache))
	for path, entries := range s.cache {
		cloned := make([]Task, len(entries))
		copy(cloned, entries)
		tasks[path] = cloned
	}

	return &Snapshot{tasks: tasks, total: s.total, created: s.now()}, nil
}

// QueueUpdate schedules a relative path for incremental reparsing.
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

// Close releases resources owned by the task index service.
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
	s.cache = nil
	s.pending = nil
	s.total = 0
	return nil
}

func (s *Service) ensureFresh() error {
	if s == nil {
		return ErrUnavailable
	}

	s.mu.RLock()
	closed := s.closed
	needsRebuild := s.cache == nil
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
	tasks, total, err := s.parseVault()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrClosed
	}

	s.cache = tasks
	s.total = total
	if s.pending != nil {
		s.pending = make(map[string]struct{})
	}
	return nil
}

func (s *Service) applyPending() error {
	rels := s.consumePending()
	if len(rels) == 0 {
		return nil
	}

	type update struct {
		path  string
		tasks []Task
	}

	updates := make([]update, 0, len(rels))
	removals := make([]string, 0)

	for _, rel := range rels {
		abs := filepath.Join(s.vault, filepath.FromSlash(rel))
		normalized := pathutil.NormalizePath(abs)
		if normalized == "" {
			continue
		}

		info, err := s.stat(normalized)
		switch {
		case err == nil:
			if info.IsDir() {
				if err := s.rebuild(); err != nil {
					return err
				}
				continue
			}
			if !strings.EqualFold(filepath.Ext(info.Name()), ".md") {
				removals = append(removals, normalized)
				continue
			}
			tasks, err := s.parseFile(normalized)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					removals = append(removals, normalized)
					continue
				}
				return err
			}
			updates = append(updates, update{path: normalized, tasks: tasks})
		case errors.Is(err, fs.ErrNotExist):
			removals = append(removals, normalized)
		default:
			return fmt.Errorf("stat %s: %w", normalized, err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrClosed
	}
	if s.cache == nil {
		s.cache = make(map[string][]Task)
	}

	newTotal := s.total
	newCount := len(s.cache)

	for _, path := range removals {
		if existing, ok := s.cache[path]; ok {
			newTotal -= len(existing)
			delete(s.cache, path)
			newCount--
		}
	}

	for _, u := range updates {
		if existing, ok := s.cache[u.path]; ok {
			newTotal -= len(existing)
			if len(u.tasks) == 0 {
				newCount--
			}
		} else if len(u.tasks) > 0 {
			newCount++
		}

		if len(u.tasks) == 0 {
			delete(s.cache, u.path)
			continue
		}

		cloned := make([]Task, len(u.tasks))
		copy(cloned, u.tasks)
		s.cache[u.path] = cloned
		newTotal += len(u.tasks)
	}

	if newTotal < 0 {
		newTotal = 0
	}
	if newCount < 0 {
		newCount = 0
	}

	if newTotal > s.maxTasks {
		return fmt.Errorf("task index size %d exceeds maximum of %d", newTotal, s.maxTasks)
	}
	if newCount > s.maxNotes {
		return fmt.Errorf("task index tracked notes %d exceeds maximum of %d", newCount, s.maxNotes)
	}

	s.total = newTotal
	return nil
}

func (s *Service) consumePending() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pending) == 0 {
		return nil
	}

	rels := make([]string, 0, len(s.pending))
	for rel := range s.pending {
		rels = append(rels, rel)
	}
	s.pending = make(map[string]struct{})
	return rels
}

func (s *Service) parseVault() (map[string][]Task, int, error) {
	if s.vault == "" {
		return nil, 0, errors.New("vault directory cannot be empty")
	}

	p := s.parserFactory(s.vault)
	if err := p.Walk(); err != nil {
		return nil, 0, err
	}

	tasks := make(map[string][]Task)
	total := 0
	for _, task := range p.TaskHandler.Tasks {
		normalized := pathutil.NormalizePath(task.Path)
		entry := toTask(task)
		tasks[normalized] = append(tasks[normalized], entry)
		total++
		if total > s.maxTasks {
			return nil, 0, fmt.Errorf("task index size %d exceeds maximum of %d", total, s.maxTasks)
		}
	}

	if len(tasks) > s.maxNotes {
		return nil, 0, fmt.Errorf("task index tracked notes %d exceeds maximum of %d", len(tasks), s.maxNotes)
	}

	return tasks, total, nil
}

func (s *Service) parseFile(path string) ([]Task, error) {
	p := s.parserFactory(path)
	if err := p.Walk(); err != nil {
		return nil, err
	}

	entries := make([]Task, 0, len(p.TaskHandler.Tasks))
	for _, task := range p.TaskHandler.Tasks {
		if !strings.EqualFold(pathutil.NormalizePath(task.Path), pathutil.NormalizePath(path)) {
			continue
		}
		entries = append(entries, toTask(task))
	}

	return entries, nil
}

func toTask(t parser.Task) Task {
	return Task{
		Path:     pathutil.NormalizePath(t.Path),
		Line:     t.Line,
		Status:   t.Status,
		Content:  t.Content,
		Metadata: cloneMetadata(t.Metadata),
	}
}

func cloneMetadata(meta parser.TaskMetadata) parser.TaskMetadata {
	cloned := parser.TaskMetadata{
		Priority:   meta.Priority,
		Owner:      meta.Owner,
		Project:    meta.Project,
		RawTokens:  nil,
		References: nil,
	}

	if meta.DueDate != nil {
		due := *meta.DueDate
		cloned.DueDate = &due
	}
	if meta.ScheduledDate != nil {
		scheduled := *meta.ScheduledDate
		cloned.ScheduledDate = &scheduled
	}
	if len(meta.References) > 0 {
		cloned.References = append([]string(nil), meta.References...)
	}
	if len(meta.RawTokens) > 0 {
		cloned.RawTokens = make(map[string]string, len(meta.RawTokens))
		for k, v := range meta.RawTokens {
			cloned.RawTokens[k] = v
		}
	}

	return cloned
}
