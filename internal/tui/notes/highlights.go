package notes

import (
	"sync"

	"github.com/Paintersrp/an/internal/search"
)

type highlightStore struct {
	mu      sync.RWMutex
	matches map[string]search.Result
}

func newHighlightStore() *highlightStore {
	return &highlightStore{matches: make(map[string]search.Result)}
}

func (s *highlightStore) setAll(entries map[string]search.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(entries) == 0 {
		s.matches = make(map[string]search.Result)
		return
	}

	s.matches = make(map[string]search.Result, len(entries))
	for path, result := range entries {
		s.matches[path] = result
	}
}

func (s *highlightStore) clear() {
	s.setAll(nil)
}

func (s *highlightStore) lookup(path string) (search.Result, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result, ok := s.matches[path]
	return result, ok
}

func (s *highlightStore) related(path string) (search.RelatedNotes, bool) {
	res, ok := s.lookup(path)
	if !ok {
		return search.RelatedNotes{}, false
	}

	related := search.RelatedNotes{}
	if len(res.Related.Outbound) > 0 {
		related.Outbound = append([]string(nil), res.Related.Outbound...)
	}
	if len(res.Related.Backlinks) > 0 {
		related.Backlinks = append([]string(nil), res.Related.Backlinks...)
	}
	return related, true
}
