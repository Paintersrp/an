package journal

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Paintersrp/an/internal/handler"
	"github.com/Paintersrp/an/internal/note"
	"github.com/Paintersrp/an/internal/templater"
	"github.com/Paintersrp/an/utils"
)

type Entry struct {
	Template string
	Path     string
	Title    string
	Date     time.Time
}

type Service struct {
	templater *templater.Templater
	handler   *handler.FileHandler
	openFunc  func(string, bool) error
}

func NewService(t *templater.Templater, h *handler.FileHandler) *Service {
	return &Service{
		templater: t,
		handler:   h,
		openFunc:  note.OpenFromPath,
	}
}

func (s *Service) EnsureEntry(templateType string, index int, tags, links []string, content string) (Entry, error) {
	if s == nil || s.templater == nil || s.handler == nil {
		return Entry{}, errors.New("journal service is not configured")
	}

	vault := s.handler.VaultDir()
	date := utils.GenerateDate(index, templateType)
	filename := fmt.Sprintf("%s-%s", templateType, date)

	n := note.NewZettelkastenNote(
		vault,
		"atoms",
		filename,
		tags,
		links,
		"",
	)

	exists, path, err := n.FileExists()
	if err != nil {
		return Entry{}, err
	}

	if !exists {
		if _, err := n.Create(templateType, s.templater, content); err != nil {
			return Entry{}, err
		}
		path = n.GetFilepath()
	}

	entry := Entry{
		Template: templateType,
		Path:     path,
		Title:    filename,
		Date:     parseDate(templateType, date),
	}
	return entry, nil
}

func (s *Service) Open(path string) error {
	if s == nil {
		return errors.New("journal service is not configured")
	}
	return s.openFunc(path, false)
}

func (s *Service) List(templateType string) ([]Entry, error) {
	if s == nil || s.handler == nil {
		return nil, errors.New("journal service is not configured")
	}

	vault := s.handler.VaultDir()
	root := filepath.Join(vault, "atoms")
	pattern := templateType + "-"

	entries := make([]Entry, 0)
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) != ".md" {
			return nil
		}
		name := strings.TrimSuffix(d.Name(), ".md")
		if !strings.HasPrefix(name, pattern) {
			return nil
		}

		suffix := strings.TrimPrefix(name, pattern)
		entry := Entry{
			Template: templateType,
			Path:     path,
			Title:    name,
			Date:     parseDate(templateType, suffix),
		}
		entries = append(entries, entry)
		return nil
	}

	if err := filepath.WalkDir(root, walkFn); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []Entry{}, nil
		}
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.After(entries[j].Date)
	})

	return entries, nil
}

func parseDate(templateType, value string) time.Time {
	formats := map[string]string{
		"day":   "20060102",
		"week":  "20060102",
		"month": "200601",
		"year":  "2006",
	}

	format, ok := formats[templateType]
	if !ok {
		format = "20060102"
	}

	parsed, err := time.Parse(format, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
