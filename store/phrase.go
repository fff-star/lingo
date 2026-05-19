package store

import (
	"sort"
	"strings"
	"sync"

	"lingo/model"
)

type jsonPhraseStore struct {
	mu       sync.RWMutex
	filePath string
	data     []model.Phrase
}

func NewJSONPhraseStore(filePath string) *jsonPhraseStore {
	return &jsonPhraseStore{filePath: filePath}
}

func (s *jsonPhraseStore) Load() ([]model.Phrase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnsafe()
}

func (s *jsonPhraseStore) Save(items []model.Phrase) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if items == nil {
		items = []model.Phrase{}
	}
	if err := writeJSON(s.filePath, items); err != nil {
		return err
	}
	s.data = items
	return nil
}

func (s *jsonPhraseStore) Add(p model.Phrase) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for _, ex := range items {
		if ex.ID == p.ID {
			return ErrExists
		}
		if strings.EqualFold(ex.Phrase, p.Phrase) {
			return ErrExists
		}
	}
	items = append(items, p)
	if err := writeJSON(s.filePath, items); err != nil {
		return err
	}
	s.data = items
	return nil
}

func (s *jsonPhraseStore) Get(idPrefix string) (*model.Phrase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if strings.HasPrefix(items[i].ID, idPrefix) || strings.EqualFold(items[i].Phrase, idPrefix) {
			return &items[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *jsonPhraseStore) Update(p model.Phrase) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == p.ID {
			items[i] = p
			if err := writeJSON(s.filePath, items); err != nil {
				return err
			}
			s.data = items
			return nil
		}
	}
	return ErrNotFound
}

func (s *jsonPhraseStore) Delete(idPrefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range items {
		if strings.HasPrefix(items[i].ID, idPrefix) || strings.EqualFold(items[i].Phrase, idPrefix) {
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

func (s *jsonPhraseStore) Search(keywords []string, tags []string) ([]model.Phrase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	var result []model.Phrase
	for _, p := range items {
		if !MatchAnyTag(p.Tags, tags) {
			continue
		}
		if len(keywords) == 0 {
			result = append(result, p)
			continue
		}
		searchText := p.Phrase + " " + p.Definition + " " + p.Type + " " +
			strings.Join(p.Synonyms, " ") + " " + strings.Join(p.Advanced, " ")
		if matchAll(searchText, keywords) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (s *jsonPhraseStore) GetAllTags() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var tags []string
	for _, p := range items {
		for _, t := range p.Tags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	sort.Strings(tags)
	return tags, nil
}

func (s *jsonPhraseStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *jsonPhraseStore) loadUnsafe() ([]model.Phrase, error) {
	if s.data != nil {
		out := make([]model.Phrase, len(s.data))
		copy(out, s.data)
		return out, nil
	}
	var items []model.Phrase
	if err := readJSON(s.filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Phrase{}
	}
	s.data = items
	return s.data, nil
}
