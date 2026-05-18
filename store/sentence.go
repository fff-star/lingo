package store

import (
	"sort"
	"strings"
	"sync"

	"lingo/model"
)

type SentenceStore struct {
	mu       sync.RWMutex
	filePath string
}

func NewSentenceStore(filePath string) *SentenceStore {
	return &SentenceStore{filePath: filePath}
}

func (s *SentenceStore) Load() ([]model.Sentence, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []model.Sentence
	if err := readJSON(s.filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Sentence{}
	}
	return items, nil
}

func (s *SentenceStore) Save(items []model.Sentence) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if items == nil {
		items = []model.Sentence{}
	}
	return writeJSON(s.filePath, items)
}

func (s *SentenceStore) Add(st model.Sentence) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for _, ex := range items {
		if ex.ID == st.ID {
			return ErrExists
		}
	}
	items = append(items, st)
	return writeJSON(s.filePath, items)
}

func (s *SentenceStore) Get(idPrefix string) (*model.Sentence, error) {
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

func (s *SentenceStore) Update(st model.Sentence) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == st.ID {
			items[i] = st
			return writeJSON(s.filePath, items)
		}
	}
	return ErrNotFound
}

func (s *SentenceStore) Delete(idPrefix string) error {
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

func (s *SentenceStore) Search(keywords []string, tags []string) ([]model.Sentence, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	var result []model.Sentence
	for _, st := range items {
		if !matchAnyTag(st.Tags, tags) {
			continue
		}
		if len(keywords) == 0 {
			result = append(result, st)
			continue
		}
		searchText := st.Text + " " + st.Translation + " " + st.Source + " " + st.Notes
		if matchAll(searchText, keywords) {
			result = append(result, st)
		}
	}
	return result, nil
}

func (s *SentenceStore) GetAllTags() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var tags []string
	for _, st := range items {
		for _, t := range st.Tags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	sort.Strings(tags)
	return tags, nil
}

func (s *SentenceStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *SentenceStore) loadUnsafe() ([]model.Sentence, error) {
	var items []model.Sentence
	if err := readJSON(s.filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Sentence{}
	}
	return items, nil
}
