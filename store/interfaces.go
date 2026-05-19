package store

import "lingo/model"

// WordStore is the interface for word persistence.
type WordStore interface {
	Add(model.Word) error
	Get(idPrefix string) (*model.Word, error)
	Update(model.Word) error
	Delete(idPrefix string) error
	Search(keywords []string, tags []string) ([]model.Word, error)
	Count() (int, error)
	GetAllTags() ([]string, error)
	AllIDs() (map[string]string, error)
	Load() ([]model.Word, error)
	LoadDue() ([]model.Word, error)
	Save([]model.Word) error
}

// PhraseStore is the interface for phrase persistence.
type PhraseStore interface {
	Add(model.Phrase) error
	Get(idPrefix string) (*model.Phrase, error)
	Update(model.Phrase) error
	Delete(idPrefix string) error
	Search(keywords []string, tags []string) ([]model.Phrase, error)
	Count() (int, error)
	GetAllTags() ([]string, error)
	Load() ([]model.Phrase, error)
	LoadDue() ([]model.Phrase, error)
}

// SentenceStore is the interface for sentence persistence.
type SentenceStore interface {
	Add(model.Sentence) error
	Get(idPrefix string) (*model.Sentence, error)
	Update(model.Sentence) error
	Delete(idPrefix string) error
	Search(keywords []string, tags []string) ([]model.Sentence, error)
	Count() (int, error)
	GetAllTags() ([]string, error)
	Load() ([]model.Sentence, error)
}

// ArticleStore is the interface for article persistence.
type ArticleStore interface {
	Add(model.Article) error
	Get(idPrefix string) (*model.Article, error)
	Update(model.Article) error
	Delete(idPrefix string) error
	Search(keywords []string, tags []string) ([]model.Article, error)
	Count() (int, error)
	GetAllTags() ([]string, error)
	Load() ([]model.Article, error)
}

// CompositionStore is the interface for composition persistence.
type CompositionStore interface {
	Add(model.Composition) error
	Get(idPrefix string) (*model.Composition, error)
	Update(model.Composition) error
	Delete(idPrefix string) error
	Search(keywords []string, tags []string) ([]model.Composition, error)
	Count() (int, error)
	GetAllTags() ([]string, error)
	Load() ([]model.Composition, error)
}

// TagStore is the interface for tag persistence.
type TagStore interface {
	Add(model.Tag) error
	Get(name string) (*model.Tag, error)
	Delete(name string) error
	Rename(oldName, newName string) error
	Load() ([]model.Tag, error)
	Save([]model.Tag) error
}

// ReviewLog is the interface for review log persistence.
type ReviewLog interface {
	Record(date string) error
	Stats(newCounts map[string]int) *ReviewStats
	TodayCount() (int, error)
	Streak() (current, longest int, err error)
}
