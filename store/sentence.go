package store

import (
	"sort"
	"strings"
	"sync"

	"lingo/model"
)

type jsonSentenceStore struct {
	mu       sync.RWMutex
	filePath string
	data     []model.Sentence
}

func NewJSONSentenceStore(filePath string) *jsonSentenceStore {
	return &jsonSentenceStore{filePath: filePath}
}

func (s *jsonSentenceStore) Load() ([]model.Sentence, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnsafe()
}

func (s *jsonSentenceStore) Save(items []model.Sentence) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if items == nil {
		items = []model.Sentence{}
	}
	if err := writeJSON(s.filePath, items); err != nil {
		return err
	}
	s.data = items
	return nil
}

func (s *jsonSentenceStore) Add(st model.Sentence) error {
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
	if err := writeJSON(s.filePath, items); err != nil {
		return err
	}
	s.data = items
	return nil
}

func (s *jsonSentenceStore) Get(idPrefix string) (*model.Sentence, error) {
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

func (s *jsonSentenceStore) Update(st model.Sentence) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == st.ID {
			items[i] = st
			if err := writeJSON(s.filePath, items); err != nil {
				return err
			}
			s.data = items
			return nil
		}
	}
	return ErrNotFound
}

func (s *jsonSentenceStore) Delete(idPrefix string) error {
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

func (s *jsonSentenceStore) Search(keywords []string, tags []string) ([]model.Sentence, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	var result []model.Sentence
	for _, st := range items {
		if !MatchAnyTag(st.Tags, tags) {
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

func (s *jsonSentenceStore) GetAllTags() ([]string, error) {
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

func (s *jsonSentenceStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *jsonSentenceStore) loadUnsafe() ([]model.Sentence, error) {
	if s.data != nil {
		out := make([]model.Sentence, len(s.data))
		copy(out, s.data)
		return out, nil
	}
	var items []model.Sentence
	if err := readJSON(s.filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Sentence{}
	}
	s.data = items
	return s.data, nil
}
