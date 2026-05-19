package store

import (
	"sort"
	"strings"
	"sync"

	"lingo/model"
)

type jsonArticleStore struct {
	mu       sync.RWMutex
	filePath string
	data     []model.Article
}

func NewJSONArticleStore(filePath string) *jsonArticleStore {
	return &jsonArticleStore{filePath: filePath}
}

func (s *jsonArticleStore) Load() ([]model.Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnsafe()
}

func (s *jsonArticleStore) Save(items []model.Article) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if items == nil {
		items = []model.Article{}
	}
	if err := writeJSON(s.filePath, items); err != nil {
		return err
	}
	s.data = items
	return nil
}

func (s *jsonArticleStore) Add(a model.Article) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for _, ex := range items {
		if ex.ID == a.ID {
			return ErrExists
		}
	}
	items = append(items, a)
	if err := writeJSON(s.filePath, items); err != nil {
		return err
	}
	s.data = items
	return nil
}

func (s *jsonArticleStore) Get(idPrefix string) (*model.Article, error) {
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

func (s *jsonArticleStore) Update(a model.Article) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == a.ID {
			items[i] = a
			if err := writeJSON(s.filePath, items); err != nil {
				return err
			}
			s.data = items
			return nil
		}
	}
	return ErrNotFound
}

func (s *jsonArticleStore) Delete(idPrefix string) error {
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

func (s *jsonArticleStore) Search(keywords []string, tags []string) ([]model.Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	var result []model.Article
	for _, a := range items {
		if !MatchAnyTag(a.Tags, tags) {
			continue
		}
		if len(keywords) == 0 {
			result = append(result, a)
			continue
		}
		searchText := a.Title + " " + a.Author + " " + a.Source + " " + a.Summary + " " + a.Content + " " + a.Notes
		if matchAll(searchText, keywords) {
			result = append(result, a)
		}
	}
	return result, nil
}

func (s *jsonArticleStore) GetAllTags() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var tags []string
	for _, a := range items {
		for _, t := range a.Tags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	sort.Strings(tags)
	return tags, nil
}

func (s *jsonArticleStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *jsonArticleStore) loadUnsafe() ([]model.Article, error) {
	if s.data != nil {
		out := make([]model.Article, len(s.data))
		copy(out, s.data)
		return out, nil
	}
	var items []model.Article
	if err := readJSON(s.filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Article{}
	}
	s.data = items
	return s.data, nil
}
