package store

import (
	"sort"
	"strings"
	"sync"

	"lingo/model"
)

type PhraseStore struct {
	mu       sync.RWMutex
	filePath string
}

func NewPhraseStore(filePath string) *PhraseStore {
	return &PhraseStore{filePath: filePath}
}

func (s *PhraseStore) Load() ([]model.Phrase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []model.Phrase
	if err := readJSON(s.filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Phrase{}
	}
	return items, nil
}

func (s *PhraseStore) Save(items []model.Phrase) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if items == nil {
		items = []model.Phrase{}
	}
	return writeJSON(s.filePath, items)
}

func (s *PhraseStore) Add(p model.Phrase) error {
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
	return writeJSON(s.filePath, items)
}

func (s *PhraseStore) Get(idPrefix string) (*model.Phrase, error) {
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

func (s *PhraseStore) Update(p model.Phrase) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == p.ID {
			items[i] = p
			return writeJSON(s.filePath, items)
		}
	}
	return ErrNotFound
}

func (s *PhraseStore) Delete(idPrefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range items {
		if strings.HasPrefix(items[i].ID, idPrefix) || strings.EqualFold(items[i].Phrase, idPrefix) {
			items = append(items[:i], items[i+1:]...)
			return writeJSON(s.filePath, items)
		}
	}
	return ErrNotFound
}

func (s *PhraseStore) Search(keywords []string, tags []string) ([]model.Phrase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	var result []model.Phrase
	for _, p := range items {
		if !matchAnyTag(p.Tags, tags) {
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

func (s *PhraseStore) GetAllTags() ([]string, error) {
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

func (s *PhraseStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadUnsafe()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *PhraseStore) loadUnsafe() ([]model.Phrase, error) {
	var items []model.Phrase
	if err := readJSON(s.filePath, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Phrase{}
	}
	return items, nil
}
