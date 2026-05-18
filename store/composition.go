package store

import (
	"sort"
	"strings"
	"sync"

	"lingo/model"
)

type CompositionStore struct {
	mu       sync.RWMutex
	filePath string
}

func NewCompositionStore(filePath string) *CompositionStore {
	return &CompositionStore{filePath: filePath}
}

func (s *CompositionStore) Load() ([]model.Composition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []model.Composition
	if err := readJSON(s.filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Composition{}
	}
	return items, nil
}

func (s *CompositionStore) Save(items []model.Composition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if items == nil {
		items = []model.Composition{}
	}
	return writeJSON(s.filePath, items)
}

func (s *CompositionStore) Add(c model.Composition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := loadCompositions(s.filePath)
	if err != nil {
		return err
	}
	for _, ex := range items {
		if ex.ID == c.ID {
			return ErrExists
		}
	}
	items = append(items, c)
	return writeJSON(s.filePath, items)
}

func (s *CompositionStore) Get(idPrefix string) (*model.Composition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := loadCompositions(s.filePath)
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

func (s *CompositionStore) Update(c model.Composition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := loadCompositions(s.filePath)
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == c.ID {
			items[i] = c
			return writeJSON(s.filePath, items)
		}
	}
	return ErrNotFound
}

func (s *CompositionStore) Delete(idPrefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := loadCompositions(s.filePath)
	if err != nil {
		return err
	}
	for i := range items {
		if strings.HasPrefix(items[i].ID, idPrefix) {
			items = append(items[:i], items[i+1:]...)
			return writeJSON(s.filePath, items)
		}
	}
	return ErrNotFound
}

func (s *CompositionStore) Search(keywords []string, tags []string) ([]model.Composition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := loadCompositions(s.filePath)
	if err != nil {
		return nil, err
	}
	var result []model.Composition
	for _, c := range items {
		if !matchAnyTag(c.Tags, tags) {
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

func (s *CompositionStore) GetAllTags() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := loadCompositions(s.filePath)
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

func (s *CompositionStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := loadCompositions(s.filePath)
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func loadCompositions(filePath string) ([]model.Composition, error) {
	var items []model.Composition
	if err := readJSON(filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Composition{}
	}
	return items, nil
}
