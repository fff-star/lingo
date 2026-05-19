package store

import (
	"sort"
	"strings"
	"sync"

	"lingo/model"
)

type jsonCompositionStore struct {
	mu       sync.RWMutex
	filePath string
	data     []model.Composition
}

func NewJSONCompositionStore(filePath string) *jsonCompositionStore {
	return &jsonCompositionStore{filePath: filePath}
}

func (s *jsonCompositionStore) Load() ([]model.Composition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnsafe()
}

func (s *jsonCompositionStore) Save(items []model.Composition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if items == nil {
		items = []model.Composition{}
	}
	if err := writeJSON(s.filePath, items); err != nil {
		return err
	}
	s.data = items
	return nil
}

func (s *jsonCompositionStore) Add(c model.Composition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for _, ex := range items {
		if ex.ID == c.ID {
			return ErrExists
		}
	}
	items = append(items, c)
	if err := writeJSON(s.filePath, items); err != nil {
		return err
	}
	s.data = items
	return nil
}

func (s *jsonCompositionStore) Get(idPrefix string) (*model.Composition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if strings.HasPrefix(items[i].ID, idPrefix) {
			return &items[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *jsonCompositionStore) Update(c model.Composition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == c.ID {
			items[i] = c
			if err := writeJSON(s.filePath, items); err != nil {
				return err
			}
			s.data = items
			return nil
		}
	}
	return ErrNotFound
}

func (s *jsonCompositionStore) Delete(idPrefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range items {
		if strings.HasPrefix(items[i].ID, idPrefix) {
			items = append(items[:i], items[i+1:]...)
			if err := writeJSON(s.filePath, items); err != nil {
				return err
			}
			s.data = items
			return nil
		}
	}
	return ErrNotFound
}

func (s *jsonCompositionStore) Search(keywords []string, tags []string) ([]model.Composition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	var result []model.Composition
	for _, c := range items {
		if !MatchAnyTag(c.Tags, tags) {
			continue
		}
		if len(keywords) == 0 {
			result = append(result, c)
			continue
		}
		searchText := c.Title + " " + c.Author + " " + c.Content + " " + c.Notes
		if c.AIAnalysis != nil {
			searchText += " " + c.AIAnalysis.Summary
		}
		if matchAll(searchText, keywords) {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *jsonCompositionStore) GetAllTags() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var tags []string
	for _, c := range items {
		for _, t := range c.Tags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	sort.Strings(tags)
	return tags, nil
}

func (s *jsonCompositionStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *jsonCompositionStore) loadUnsafe() ([]model.Composition, error) {
	if s.data != nil {
		out := make([]model.Composition, len(s.data))
		copy(out, s.data)
		return out, nil
	}
	var items []model.Composition
	if err := readJSON(s.filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Composition{}
	}
	s.data = items
	return s.data, nil
}
