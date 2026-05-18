package store

import (
	"sort"
	"strings"
	"sync"

	"lingo/model"
)

type ArticleStore struct {
	mu       sync.RWMutex
	filePath string
}

func NewArticleStore(filePath string) *ArticleStore {
	return &ArticleStore{filePath: filePath}
}

func (s *ArticleStore) Load() ([]model.Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []model.Article
	if err := readJSON(s.filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Article{}
	}
	return items, nil
}

func (s *ArticleStore) Save(items []model.Article) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if items == nil {
		items = []model.Article{}
	}
	return writeJSON(s.filePath, items)
}

func (s *ArticleStore) Add(a model.Article) error {
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
	return writeJSON(s.filePath, items)
}

func (s *ArticleStore) Get(idPrefix string) (*model.Article, error) {
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

func (s *ArticleStore) Update(a model.Article) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == a.ID {
			items[i] = a
			return writeJSON(s.filePath, items)
		}
	}
	return ErrNotFound
}

func (s *ArticleStore) Delete(idPrefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
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

func (s *ArticleStore) Search(keywords []string, tags []string) ([]model.Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	var result []model.Article
	for _, a := range items {
		if !matchAnyTag(a.Tags, tags) {
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

func (s *ArticleStore) GetAllTags() ([]string, error) {
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

func (s *ArticleStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *ArticleStore) loadUnsafe() ([]model.Article, error) {
	var items []model.Article
	if err := readJSON(s.filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Article{}
	}
	return items, nil
}
