package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"lingo/model"
)

func tempPath() string {
	dir := os.TempDir()
	return filepath.Join(dir, "lingo_test_"+NewID("test")+".json")
}

func TestWordStoreCRUD(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	s := NewWordStore(path)

	// Add.
	now := time.Now().UTC()
	w := model.Word{
		ID:          NewID("wd"),
		Word:        "ephemeral",
		Definitions: []model.Definition{{Pos: "adj", Meaning: "short-lived"}},
		Synonyms:    []string{},
		Advanced:    []string{},
		Examples:    []string{},
		Inflections: []model.Inflection{},
		Tags:        []string{},
		Stability:   0,
		Difficulty:  5,
		NextReviewAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.Add(w); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Duplicate word should fail.
	if err := s.Add(w); err != ErrExists {
		t.Errorf("duplicate Add = %v, want ErrExists", err)
	}

	// Get.
	got, err := s.Get(w.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Word != "ephemeral" {
		t.Errorf("Get word = %q, want %q", got.Word, "ephemeral")
	}

	// Get by word.
	got, err = s.Get("ephemeral")
	if err != nil {
		t.Fatalf("Get by word: %v", err)
	}

	// Update.
	w.Word = "updated"
	if err := s.Update(w); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = s.Get(w.ID)
	if got.Word != "updated" {
		t.Errorf("after update = %q, want %q", got.Word, "updated")
	}

	// Delete.
	if err := s.Delete(w.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(w.ID); err != ErrNotFound {
		t.Errorf("after delete Get = %v, want ErrNotFound", err)
	}
}

func TestWordStoreSearch(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	s := NewWordStore(path)

	now := time.Now().UTC()
	s.Add(model.Word{
		ID: NewID("wd"), Word: "ephemeral",
		Definitions: []model.Definition{{Meaning: "short-lived"}},
		Synonyms:    []string{},
		Advanced:    []string{},
		Examples:    []string{},
		Inflections: []model.Inflection{},
		Tags:        []string{"gre"},
		NextReviewAt: now, CreatedAt: now, UpdatedAt: now,
	})
	s.Add(model.Word{
		ID: NewID("wd"), Word: "ubiquitous",
		Definitions: []model.Definition{{Meaning: "everywhere"}},
		Synonyms:    []string{},
		Advanced:    []string{},
		Examples:    []string{},
		Inflections: []model.Inflection{},
		Tags:        []string{"toefl"},
		NextReviewAt: now, CreatedAt: now, UpdatedAt: now,
	})

	// Keyword search.
	results, _ := s.Search([]string{"short"}, nil)
	if len(results) != 1 {
		t.Errorf("search 'short': %d results, want 1", len(results))
	}

	// Tag filter.
	results, _ = s.Search(nil, []string{"gre"})
	if len(results) != 1 {
		t.Errorf("tag 'gre': %d results, want 1", len(results))
	}

	// Count.
	count, _ := s.Count()
	if count != 2 {
		t.Errorf("Count = %d, want 2", count)
	}
}

func TestTagStore(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	s := NewTagStore(path)

	tag := model.Tag{ID: NewID("tag"), Name: "gre", Color: "#ff6b6b"}
	if err := s.Add(tag); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Rename.
	if err := s.Rename("gre", "GRE"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	got, _ := s.Get("GRE")
	if got.Name != "GRE" {
		t.Errorf("after rename = %q, want GRE", got.Name)
	}

	// Delete.
	if err := s.Delete("GRE"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestNewID(t *testing.T) {
	id1 := NewID("wd")
	id2 := NewID("wd")
	if id1 == id2 {
		t.Error("two IDs should be different")
	}
	if len(id1) != 15 { // "wd_" + 12 hex chars
		t.Errorf("id length = %d, want 15", len(id1))
	}
}

func TestMatchAll(t *testing.T) {
	tests := []struct {
		s        string
		keywords []string
		want     bool
	}{
		{"hello world", []string{"hello"}, true},
		{"hello world", []string{"HELLO"}, true},
		{"hello world", []string{"hello", "world"}, true},
		{"hello world", []string{"hello", "missing"}, false},
		{"hello world", []string{}, true},
		{"", []string{"x"}, false},
	}
	for _, tt := range tests {
		got := matchAll(tt.s, tt.keywords)
		if got != tt.want {
			t.Errorf("matchAll(%q, %v) = %v, want %v", tt.s, tt.keywords, got, tt.want)
		}
	}
}

func TestMatchAnyTag(t *testing.T) {
	tags := []string{"gre", "toefl"}
	if !matchAnyTag(tags, nil) {
		t.Error("nil filter should match")
	}
	if !matchAnyTag(tags, []string{}) {
		t.Error("empty filter should match")
	}
	if !matchAnyTag(tags, []string{"gre"}) {
		t.Error("should match 'gre'")
	}
	if matchAnyTag(tags, []string{"ielts"}) {
		t.Error("should not match 'ielts'")
	}
	if !matchAnyTag(tags, []string{"gre", "ielts"}) {
		t.Error("should match when one of many matches")
	}
}

func TestPhraseStoreCRUD(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	s := NewPhraseStore(path)

	now := time.Now().UTC()
	p := model.Phrase{
		ID:           NewID("ph"),
		Phrase:       "in the long run",
		Type:         "idiom",
		Words:        []string{"in", "the", "long", "run"},
		Definition:   "eventually",
		Examples:     []string{},
		Synonyms:     []string{},
		Advanced:     []string{},
		Tags:         []string{"writing"},
		NextReviewAt: now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.Add(p); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(p); err != ErrExists {
		t.Errorf("duplicate Add = %v, want ErrExists", err)
	}

	got, _ := s.Get(p.ID)
	if got.Phrase != "in the long run" {
		t.Errorf("Get phrase = %q", got.Phrase)
	}

	// Get by phrase text.
	got, _ = s.Get("in the long run")
	if got == nil {
		t.Error("Get by phrase text failed")
	}

	p.Definition = "over time"
	if err := s.Update(p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := s.Delete(p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(p.ID); err != ErrNotFound {
		t.Errorf("after delete Get = %v, want ErrNotFound", err)
	}
}

func TestPhraseStoreSearch(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	s := NewPhraseStore(path)
	now := time.Now().UTC()
	s.Add(model.Phrase{
		ID: NewID("ph"), Phrase: "in the long run", Definition: "eventually",
		Tags: []string{"writing"}, Examples: []string{}, Synonyms: []string{}, Advanced: []string{},
		NextReviewAt: now, CreatedAt: now, UpdatedAt: now,
	})
	s.Add(model.Phrase{
		ID: NewID("ph"), Phrase: "by the way", Definition: "incidentally",
		Tags: []string{"speaking"}, Examples: []string{}, Synonyms: []string{}, Advanced: []string{},
		NextReviewAt: now, CreatedAt: now, UpdatedAt: now,
	})

	results, _ := s.Search([]string{"eventually"}, nil)
	if len(results) != 1 {
		t.Errorf("search 'eventually': %d results, want 1", len(results))
	}

	results, _ = s.Search(nil, []string{"speaking"})
	if len(results) != 1 {
		t.Errorf("tag 'speaking': %d results, want 1", len(results))
	}
}

func TestReviewLogRecord(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	rl := NewReviewLog(path)

	today := time.Now().UTC().Format("2006-01-02")
	if err := rl.Record(today); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := rl.Record(today); err != nil {
		t.Fatalf("Record 2: %v", err)
	}

	stats := rl.Stats(map[string]int{})
	found := false
	for i, label := range stats.Labels {
		// Today should be one of the last 15 days (most recent).
		if label == time.Now().UTC().Format("01-02") {
			if stats.Reviews[i] != 2 {
				t.Errorf("reviews for today = %d, want 2", stats.Reviews[i])
			}
			found = true
		}
	}
	if !found {
		t.Log("today not in range (may need 15 days of data)")
	}
}

func TestNewCountsByDate(t *testing.T) {
	now := time.Now().UTC()
	yesterday := now.AddDate(0, 0, -1)

	dates := []time.Time{now, now, yesterday}
	counts := NewCountsByDate(dates)

	todayKey := now.Format("2006-01-02")
	if counts[todayKey] != 2 {
		t.Errorf("today count = %d, want 2", counts[todayKey])
	}
	yesterdayKey := yesterday.Format("2006-01-02")
	if counts[yesterdayKey] != 1 {
		t.Errorf("yesterday count = %d, want 1", counts[yesterdayKey])
	}
}

func TestMergeCounts(t *testing.T) {
	a := map[string]int{"2026-05-01": 2, "2026-05-02": 1}
	b := map[string]int{"2026-05-01": 3, "2026-05-03": 4}
	merged := MergeCounts(a, b)
	if merged["2026-05-01"] != 5 {
		t.Errorf("merged 05-01 = %d, want 5", merged["2026-05-01"])
	}
	if merged["2026-05-02"] != 1 {
		t.Errorf("merged 05-02 = %d, want 1", merged["2026-05-02"])
	}
	if merged["2026-05-03"] != 4 {
		t.Errorf("merged 05-03 = %d, want 4", merged["2026-05-03"])
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]int{"c": 3, "a": 1, "b": 2}
	keys := SortedKeys(m)
	if len(keys) != 3 {
		t.Fatalf("len = %d, want 3", len(keys))
	}
	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("keys = %v, want [a b c]", keys)
	}
}
