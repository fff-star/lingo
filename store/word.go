package store

import (
	"sort"
	"strings"
	"sync"

	"lingo/model"
)

type WordStore struct {
	mu       sync.RWMutex
	filePath string
}

func NewWordStore(filePath string) *WordStore {
	return &WordStore{filePath: filePath}
}

func (s *WordStore) Load() ([]model.Word, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var words []model.Word
	if err := readJSON(s.filePath, &words); err != nil {
		return nil, err
	}
	if words == nil {
		words = []model.Word{}
	}
	return words, nil
}

func (s *WordStore) Save(words []model.Word) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if words == nil {
		words = []model.Word{}
	}
	return writeJSON(s.filePath, words)
}

func (s *WordStore) Add(w model.Word) error {
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
	return writeJSON(s.filePath, words)
}

func (s *WordStore) Get(idPrefix string) (*model.Word, error) {
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

func (s *WordStore) Update(w model.Word) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range words {
		if words[i].ID == w.ID {
			words[i] = w
			return writeJSON(s.filePath, words)
		}
	}
	return ErrNotFound
}

func (s *WordStore) Delete(idPrefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range words {
		if strings.HasPrefix(words[i].ID, idPrefix) || strings.EqualFold(words[i].Word, idPrefix) {
			words = append(words[:i], words[i+1:]...)
			return writeJSON(s.filePath, words)
		}
	}
	return ErrNotFound
}

func (s *WordStore) Search(keywords []string, tags []string) ([]model.Word, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	var result []model.Word
	for _, w := range words {
		if !matchAnyTag(w.Tags, tags) {
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

func (s *WordStore) AllIDs() (map[string]string, error) {
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

func (s *WordStore) GetAllTags() ([]string, error) {
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

func (s *WordStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	words, err := s.loadUnsafe()
	if err != nil {
		return 0, err
	}
	return len(words), nil
}

func (s *WordStore) loadUnsafe() ([]model.Word, error) {
	var words []model.Word
	if err := readJSON(s.filePath, &words); err != nil {
		return nil, err
	}
	if words == nil {
		words = []model.Word{}
	}
	return words, nil
}
