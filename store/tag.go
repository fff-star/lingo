package store

import (
	"strings"
	"sync"

	"lingo/model"
)

type jsonTagStore struct {
	mu       sync.RWMutex
	filePath string
	data     []model.Tag
}

func NewJSONTagStore(filePath string) *jsonTagStore {
	return &jsonTagStore{filePath: filePath}
}

func (s *jsonTagStore) Load() ([]model.Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnsafe()
}

func (s *jsonTagStore) Save(tags []model.Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = []model.Tag{}
	}
	if err := writeJSON(s.filePath, tags); err != nil {
		return err
	}
	s.data = tags
	return nil
}

func (s *jsonTagStore) Add(tag model.Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tags, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for _, t := range tags {
		if t.ID == tag.ID || strings.EqualFold(t.Name, tag.Name) {
			return ErrExists
		}
	}
	tags = append(tags, tag)
	if err := writeJSON(s.filePath, tags); err != nil {
		return err
	}
	s.data = tags
	return nil
}

func (s *jsonTagStore) Get(name string) (*model.Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tags, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}
	for i := range tags {
		if strings.EqualFold(tags[i].Name, name) || strings.HasPrefix(tags[i].ID, name) {
			return &tags[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *jsonTagStore) Rename(oldName, newName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tags, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	found := false
	for i := range tags {
		if strings.EqualFold(tags[i].Name, oldName) {
			tags[i].Name = newName
			found = true
			break
		}
	}
	if !found {
		return ErrNotFound
	}
	if err := writeJSON(s.filePath, tags); err != nil {
		return err
	}
	s.data = tags
	return nil
}

func (s *jsonTagStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tags, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range tags {
		if strings.EqualFold(tags[i].Name, name) || strings.HasPrefix(tags[i].ID, name) {
			tags = append(tags[:i], tags[i+1:]...)
			if err := writeJSON(s.filePath, tags); err != nil {
				return err
			}
			s.data = tags
			return nil
		}
	}
	return ErrNotFound
}

func (s *jsonTagStore) loadUnsafe() ([]model.Tag, error) {
	if s.data != nil {
		out := make([]model.Tag, len(s.data))
		copy(out, s.data)
		return out, nil
	}
	var tags []model.Tag
	if err := readJSON(s.filePath, &tags); err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []model.Tag{}
	}
	s.data = tags
	return s.data, nil
}
