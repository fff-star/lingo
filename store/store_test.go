package store

import (
	"database/sql"
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

func sqliteDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateTables(db); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// ---- JSON-backed tests ----

func TestJSONWordStoreCRUD(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	s := NewJSONWordStore(path)

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

	if err := s.Add(w); err != ErrExists {
		t.Errorf("duplicate Add = %v, want ErrExists", err)
	}

	got, err := s.Get(w.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Word != "ephemeral" {
		t.Errorf("Get word = %q, want %q", got.Word, "ephemeral")
	}

	got, err = s.Get("ephemeral")
	if err != nil {
		t.Fatalf("Get by word: %v", err)
	}

	w.Word = "updated"
	if err := s.Update(w); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = s.Get(w.ID)
	if got.Word != "updated" {
		t.Errorf("after update = %q, want %q", got.Word, "updated")
	}

	if err := s.Delete(w.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(w.ID); err != ErrNotFound {
		t.Errorf("after delete Get = %v, want ErrNotFound", err)
	}
}

func TestJSONWordStoreSearch(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	s := NewJSONWordStore(path)

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

	results, _ := s.Search([]string{"short"}, nil)
	if len(results) != 1 {
		t.Errorf("search 'short': %d results, want 1", len(results))
	}

	results, _ = s.Search(nil, []string{"gre"})
	if len(results) != 1 {
		t.Errorf("tag 'gre': %d results, want 1", len(results))
	}

	count, _ := s.Count()
	if count != 2 {
		t.Errorf("Count = %d, want 2", count)
	}
}

func TestJSONTagStore(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	s := NewJSONTagStore(path)

	tag := model.Tag{ID: NewID("tag"), Name: "gre", Color: "#ff6b6b"}
	if err := s.Add(tag); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := s.Rename("gre", "GRE"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	got, _ := s.Get("GRE")
	if got.Name != "GRE" {
		t.Errorf("after rename = %q, want GRE", got.Name)
	}

	if err := s.Delete("GRE"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestJSONPhraseStoreCRUD(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	s := NewJSONPhraseStore(path)

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

func TestJSONPhraseStoreSearch(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	s := NewJSONPhraseStore(path)
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

func TestJSONReviewLogRecord(t *testing.T) {
	path := tempPath()
	defer os.Remove(path)

	rl := NewJSONReviewLog(path)

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

// ---- SQLite-backed tests ----

func TestSQLiteWordStoreCRUD(t *testing.T) {
	db := sqliteDB(t)
	s := NewWordStore(db)

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

	if err := s.Add(w); err != ErrExists {
		t.Errorf("duplicate Add = %v, want ErrExists", err)
	}

	got, err := s.Get(w.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Word != "ephemeral" {
		t.Errorf("Get word = %q, want %q", got.Word, "ephemeral")
	}

	got, err = s.Get("ephemeral")
	if err != nil {
		t.Fatalf("Get by word: %v", err)
	}

	w.Word = "updated"
	if err := s.Update(w); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = s.Get(w.ID)
	if got.Word != "updated" {
		t.Errorf("after update = %q, want %q", got.Word, "updated")
	}

	if err := s.Delete(w.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(w.ID); err != ErrNotFound {
		t.Errorf("after delete Get = %v, want ErrNotFound", err)
	}
}

func TestSQLiteWordStoreSearch(t *testing.T) {
	db := sqliteDB(t)
	s := NewWordStore(db)

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

	results, _ := s.Search([]string{"short"}, nil)
	if len(results) != 1 {
		t.Errorf("search 'short': %d results, want 1", len(results))
	}

	results, _ = s.Search(nil, []string{"gre"})
	if len(results) != 1 {
		t.Errorf("tag 'gre': %d results, want 1", len(results))
	}

	count, _ := s.Count()
	if count != 2 {
		t.Errorf("Count = %d, want 2", count)
	}

	ids, _ := s.AllIDs()
	if len(ids) != 2 {
		t.Errorf("AllIDs = %d, want 2", len(ids))
	}
}

func TestSQLitePhraseStore(t *testing.T) {
	db := sqliteDB(t)
	s := NewPhraseStore(db)

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

	got, _ := s.Get(p.ID)
	if got.Phrase != "in the long run" {
		t.Errorf("Get phrase = %q", got.Phrase)
	}

	if err := s.Delete(p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestSQLiteTagStore(t *testing.T) {
	db := sqliteDB(t)
	s := NewTagStore(db)

	tag := model.Tag{ID: NewID("tag"), Name: "gre", Color: "#ff6b6b"}
	if err := s.Add(tag); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := s.Rename("gre", "GRE"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	got, _ := s.Get("GRE")
	if got.Name != "GRE" {
		t.Errorf("after rename = %q, want GRE", got.Name)
	}

	if err := s.Delete("GRE"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestSQLiteReviewLog(t *testing.T) {
	db := sqliteDB(t)
	rl := NewReviewLog(db)

	today := time.Now().UTC().Format("2006-01-02")
	if err := rl.Record(today); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := rl.Record(today); err != nil {
		t.Fatalf("Record 2: %v", err)
	}

	stats := rl.Stats(map[string]int{})
	for i, label := range stats.Labels {
		if label == time.Now().UTC().Format("01-02") {
			if stats.Reviews[i] != 2 {
				t.Errorf("reviews for today = %d, want 2", stats.Reviews[i])
			}
		}
	}
}

// ---- Utility tests ----

func TestNewID(t *testing.T) {
	id1 := NewID("wd")
	id2 := NewID("wd")
	if id1 == id2 {
		t.Error("two IDs should be different")
	}
	if len(id1) != 15 {
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
	if !MatchAnyTag(tags, nil) {
		t.Error("nil filter should match")
	}
	if !MatchAnyTag(tags, []string{}) {
		t.Error("empty filter should match")
	}
	if !MatchAnyTag(tags, []string{"gre"}) {
		t.Error("should match 'gre'")
	}
	if MatchAnyTag(tags, []string{"ielts"}) {
		t.Error("should not match 'ielts'")
	}
	if !MatchAnyTag(tags, []string{"gre", "ielts"}) {
		t.Error("should match when one of many matches")
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

// ---- String slice helpers ----

func TestHasString(t *testing.T) {
	tests := []struct {
		slice  []string
		target string
		want   bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "x", false},
		{nil, "x", false},
		{[]string{"x"}, "x", true},
	}
	for _, tt := range tests {
		got := HasString(tt.slice, tt.target)
		if got != tt.want {
			t.Errorf("HasString(%v, %q) = %v, want %v", tt.slice, tt.target, got, tt.want)
		}
	}
}

func TestRemoveString(t *testing.T) {
	tests := []struct {
		slice  []string
		target string
		want   []string
	}{
		{[]string{"a", "b", "c"}, "b", []string{"a", "c"}},
		{[]string{"a", "b", "c"}, "d", []string{"a", "b", "c"}},
		{[]string{"x"}, "x", []string{}},
		{[]string{}, "x", []string{}},
	}
	for _, tt := range tests {
		got := RemoveString(tt.slice, tt.target)
		if len(got) != len(tt.want) {
			t.Errorf("RemoveString(%v, %q) len = %d, want %d", tt.slice, tt.target, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("RemoveString(%v, %q)[%d] = %q, want %q", tt.slice, tt.target, i, got[i], tt.want[i])
			}
		}
	}
}

// ---- SQLite Sentence / Article / Composition CRUD ----

func TestSQLiteSentenceCRUD(t *testing.T) {
	db := sqliteDB(t)
	s := NewSentenceStore(db)

	now := time.Now().UTC()
	st := model.Sentence{
		ID:          NewID("st"),
		Text:        "The quick brown fox jumps over the lazy dog.",
		Source:      "typing practice",
		Translation: "敏捷的棕狐狸跳过了懒狗",
		Tags:        []string{"pangram"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.Add(st); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.Get(st.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Text != st.Text {
		t.Errorf("text = %q, want %q", got.Text, st.Text)
	}
	if got.Translation != st.Translation {
		t.Errorf("translation = %q", got.Translation)
	}

	// Search by keyword.
	results, _ := s.Search([]string{"fox"}, nil)
	if len(results) != 1 {
		t.Errorf("search 'fox': %d results, want 1", len(results))
	}

	// Search by tag.
	results, _ = s.Search(nil, []string{"pangram"})
	if len(results) != 1 {
		t.Errorf("tag 'pangram': %d results, want 1", len(results))
	}

	// Update.
	st.Text = "updated text"
	if err := s.Update(st); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = s.Get(st.ID)
	if got.Text != "updated text" {
		t.Errorf("after update = %q", got.Text)
	}

	// Delete.
	if err := s.Delete(st.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(st.ID); err != ErrNotFound {
		t.Errorf("after delete = %v, want ErrNotFound", err)
	}
}

func TestSQLiteArticleCRUD(t *testing.T) {
	db := sqliteDB(t)
	s := NewArticleStore(db)

	now := time.Now().UTC()
	a := model.Article{
		ID:      NewID("ar"),
		Title:   "Climate Change Report",
		Author:  "IPCC",
		Content: "Global temperatures continue to rise at an unprecedented rate.",
		Tags:    []string{"environment", "science"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.Add(a); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.Get(a.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != a.Title {
		t.Errorf("title = %q, want %q", got.Title, a.Title)
	}
	if got.Author != "IPCC" {
		t.Errorf("author = %q", got.Author)
	}

	// Search by content keyword.
	results, _ := s.Search([]string{"temperature"}, nil)
	if len(results) != 1 {
		t.Errorf("search 'temperature': %d results, want 1", len(results))
	}

	// Search by tag.
	results, _ = s.Search(nil, []string{"science"})
	if len(results) != 1 {
		t.Errorf("tag 'science': %d results, want 1", len(results))
	}

	// Update.
	a.Title = "Updated Climate Report"
	if err := s.Update(a); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Delete.
	if err := s.Delete(a.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestSQLiteCompositionCRUD(t *testing.T) {
	db := sqliteDB(t)
	s := NewCompositionStore(db)

	now := time.Now().UTC()
	c := model.Composition{
		ID:      NewID("cp"),
		Title:   "My Summer Vacation",
		Author:  "student",
		Content: "Last summer I went to the beach and learned to surf.",
		Tags:    []string{"narrative"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.Add(c); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.Get(c.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != c.Title {
		t.Errorf("title = %q, want %q", got.Title, c.Title)
	}

	// Search by content keyword.
	results, _ := s.Search([]string{"beach"}, nil)
	if len(results) != 1 {
		t.Errorf("search 'beach': %d results, want 1", len(results))
	}

	// Update.
	c.Content = "updated content"
	if err := s.Update(c); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Delete.
	if err := s.Delete(c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// ---- LoadDue / GetAllTags / Streak ----

func TestSQLiteWordLoadDue(t *testing.T) {
	db := sqliteDB(t)
	s := NewWordStore(db)

	now := time.Now().UTC()
	past := now.Add(-24 * time.Hour)
	future := now.Add(7 * 24 * time.Hour)

	// Word due now (next_review_at in past).
	dueWord := model.Word{
		ID: NewID("wd"), Word: "due-now",
		Definitions: []model.Definition{}, Synonyms: []string{}, Advanced: []string{},
		Examples: []string{}, Inflections: []model.Inflection{}, Tags: []string{},
		NextReviewAt: past, CreatedAt: now, UpdatedAt: now,
	}
	// Word NOT due (next_review_at in future).
	futureWord := model.Word{
		ID: NewID("wd"), Word: "not-due",
		Definitions: []model.Definition{}, Synonyms: []string{}, Advanced: []string{},
		Examples: []string{}, Inflections: []model.Inflection{}, Tags: []string{},
		NextReviewAt: future, CreatedAt: now, UpdatedAt: now,
	}
	// Word never reviewed (zero sentinel).
	newWord := model.Word{
		ID: NewID("wd"), Word: "never-reviewed",
		Definitions: []model.Definition{}, Synonyms: []string{}, Advanced: []string{},
		Examples: []string{}, Inflections: []model.Inflection{}, Tags: []string{},
		NextReviewAt: time.Time{}, CreatedAt: now, UpdatedAt: now,
	}

	s.Add(dueWord)
	s.Add(futureWord)
	s.Add(newWord)

	due, err := s.LoadDue()
	if err != nil {
		t.Fatalf("LoadDue: %v", err)
	}
	// Should return due-now and never-reviewed, but NOT not-due.
	if len(due) != 2 {
		t.Errorf("LoadDue count = %d, want 2 (past + zero-time)", len(due))
		for _, w := range due {
			t.Logf("  got: %s (next=%s)", w.Word, w.NextReviewAt)
		}
	}
}

func TestSQLiteWordGetAllTags(t *testing.T) {
	db := sqliteDB(t)
	s := NewWordStore(db)

	now := time.Now().UTC()
	s.Add(model.Word{
		ID: NewID("wd"), Word: "w1",
		Definitions: []model.Definition{}, Synonyms: []string{}, Advanced: []string{},
		Examples: []string{}, Inflections: []model.Inflection{}, Tags: []string{"gre", "abstract"},
		NextReviewAt: now, CreatedAt: now, UpdatedAt: now,
	})
	s.Add(model.Word{
		ID: NewID("wd"), Word: "w2",
		Definitions: []model.Definition{}, Synonyms: []string{}, Advanced: []string{},
		Examples: []string{}, Inflections: []model.Inflection{}, Tags: []string{"toefl", "gre"},
		NextReviewAt: now, CreatedAt: now, UpdatedAt: now,
	})

	tags, err := s.GetAllTags()
	if err != nil {
		t.Fatalf("GetAllTags: %v", err)
	}
	if len(tags) != 3 {
		t.Errorf("GetAllTags count = %d, want 3 unique tags", len(tags))
	}
	// Verify uniqueness.
	seen := make(map[string]bool)
	for _, tag := range tags {
		if seen[tag] {
			t.Errorf("duplicate tag %q", tag)
		}
		seen[tag] = true
	}
}

func TestSQLiteReviewLogStreak(t *testing.T) {
	db := sqliteDB(t)
	rl := NewReviewLog(db)

	now := time.Now().UTC()

	// Record reviews for today and the last 3 consecutive days.
	for i := 0; i < 4; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		if err := rl.Record(date); err != nil {
			t.Fatalf("Record %s: %v", date, err)
		}
	}

	current, longest, err := rl.Streak()
	if err != nil {
		t.Fatalf("Streak: %v", err)
	}
	if current < 4 {
		t.Errorf("current streak = %d, want at least 4", current)
	}
	if longest < 4 {
		t.Errorf("longest streak = %d, want at least 4", longest)
	}

	// TodayCount should be >= 1 (today was recorded).
	todayCount, err := rl.TodayCount()
	if err != nil {
		t.Fatalf("TodayCount: %v", err)
	}
	if todayCount < 1 {
		t.Errorf("todayCount = %d, want >= 1", todayCount)
	}
}

func TestSQLiteReviewLogEmptyStreak(t *testing.T) {
	db := sqliteDB(t)
	rl := NewReviewLog(db)

	current, longest, err := rl.Streak()
	if err != nil {
		t.Fatalf("Streak: %v", err)
	}
	if current != 0 {
		t.Errorf("empty streak current = %d, want 0", current)
	}
	if longest != 0 {
		t.Errorf("empty streak longest = %d, want 0", longest)
	}

	count, _ := rl.TodayCount()
	if count != 0 {
		t.Errorf("empty todayCount = %d, want 0", count)
	}
}

func TestSQLiteReviewLogStats(t *testing.T) {
	db := sqliteDB(t)
	rl := NewReviewLog(db)

	now := time.Now().UTC()
	for i := 0; i < 15; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		rl.Record(date)
	}

	stats := rl.Stats(map[string]int{})
	if len(stats.Labels) != 15 {
		t.Errorf("Stats labels = %d, want 15", len(stats.Labels))
	}
	if len(stats.Reviews) != 15 {
		t.Errorf("Stats reviews = %d, want 15", len(stats.Reviews))
	}
}
