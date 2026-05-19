package store

import (
	"sort"
	"strings"
	"sync"

	"lingo/model"
)

type jsonWordStore struct {
	mu       sync.RWMutex
	filePath string
	data     []model.Word
}

func NewJSONWordStore(filePath string) *jsonWordStore {
	return &jsonWordStore{filePath: filePath}
}

func (s *jsonWordStore) Load() ([]model.Word, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnsafe()
}

func (s *jsonWordStore) Save(words []model.Word) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if words == nil {
		words = []model.Word{}
	}
	if err := writeJSON(s.filePath, words); err != nil {
		return err
	}
	s.data = words
	return nil
}

func (s *jsonWordStore) Add(w model.Word) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for _, ex := range words {
		if ex.ID == w.ID {
			return ErrExists
		}
		if strings.EqualFold(ex.Word, w.Word) {
			return ErrExists
		}
	}
	words = append(words, w)
	if err := writeJSON(s.filePath, words); err != nil {
		return err
	}
	s.data = words
	return nil
}

func (s *jsonWordStore) Get(idPrefix string) (*model.Word, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	for i := range words {
		if strings.HasPrefix(words[i].ID, idPrefix) || strings.EqualFold(words[i].Word, idPrefix) {
			return &words[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *jsonWordStore) Update(w model.Word) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range words {
		if words[i].ID == w.ID {
			words[i] = w
			if err := writeJSON(s.filePath, words); err != nil {
				return err
			}
			s.data = words
			return nil
		}
	}
	return ErrNotFound
}

func (s *jsonWordStore) Delete(idPrefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range words {
		if strings.HasPrefix(words[i].ID, idPrefix) || strings.EqualFold(words[i].Word, idPrefix) {
			words = append(words[:i], words[i+1:]...)
			if err := writeJSON(s.filePath, words); err != nil {
				return err
			}
			s.data = words
			return nil
		}
	}
	return ErrNotFound
}

func (s *jsonWordStore) Search(keywords []string, tags []string) ([]model.Word, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	var result []model.Word
	for _, w := range words {
		if !MatchAnyTag(w.Tags, tags) {
			continue
		}
		if len(keywords) == 0 {
			result = append(result, w)
			continue
		}
		searchText := w.Word + " " + strings.Join(w.Synonyms, " ") + " " + strings.Join(w.Advanced, " ")
		for _, d := range w.Definitions {
			searchText += " " + d.Meaning
		}
		if matchAll(searchText, keywords) {
			result = append(result, w)
		}
	}
	return result, nil
}

func (s *jsonWordStore) AllIDs() (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, w := range words {
		m[w.ID] = w.Word
	}
	return m, nil
}

func (s *jsonWordStore) GetAllTags() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var tags []string
	for _, w := range words {
		for _, t := range w.Tags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	sort.Strings(tags)
	return tags, nil
}

func (s *jsonWordStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return 0, err
	}
	return len(words), nil
}

func (s *jsonWordStore) loadUnsafe() ([]model.Word, error) {
	if s.data != nil {
		out := make([]model.Word, len(s.data))
		copy(out, s.data)
		return out, nil
	}
	var words []model.Word
	if err := readJSON(s.filePath, &words); err != nil {
		return nil, err
	}
	if words == nil {
		words = []model.Word{}
	}
	s.data = words
	return s.data, nil
}
