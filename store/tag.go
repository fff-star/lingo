package store

import (
	"strings"
	"sync"

	"lingo/model"
)

type TagStore struct {
	mu       sync.RWMutex
	filePath string
}

func NewTagStore(filePath string) *TagStore {
	return &TagStore{filePath: filePath}
}

func (s *TagStore) Load() ([]model.Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var tags []model.Tag
	if err := readJSON(s.filePath, &tags); err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []model.Tag{}
	}
	return tags, nil
}

func (s *TagStore) Save(tags []model.Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = []model.Tag{}
	}
	return writeJSON(s.filePath, tags)
}

func (s *TagStore) Add(tag model.Tag) error {
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
	return writeJSON(s.filePath, tags)
}

func (s *TagStore) Get(name string) (*model.Tag, error) {
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

func (s *TagStore) Rename(oldName, newName string) error {
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
	return writeJSON(s.filePath, tags)
}

func (s *TagStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tags, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	for i := range tags {
		if strings.EqualFold(tags[i].Name, name) || strings.HasPrefix(tags[i].ID, name) {
			tags = append(tags[:i], tags[i+1:]...)
			return writeJSON(s.filePath, tags)
		}
	}
	return ErrNotFound
}

func (s *TagStore) loadUnsafe() ([]model.Tag, error) {
	var tags []model.Tag
	if err := readJSON(s.filePath, &tags); err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []model.Tag{}
	}
	return tags, nil
}
