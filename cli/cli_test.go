package cli

import (
	"bytes"
	"database/sql"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"lingo/model"
	"lingo/store"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func sqliteDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func setupTestDB(t *testing.T) {
	t.Helper()
	db := sqliteDB(t)
	if err := store.CreateTables(db); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	InitWord(store.NewWordStore(db))
	InitPhrase(store.NewPhraseStore(db))
	InitSentence(store.NewSentenceStore(db))
	InitArticle(store.NewArticleStore(db))
	InitComp(store.NewCompositionStore(db))
	InitTag(store.NewTagStore(db))
}

func addTestWord(t *testing.T) model.Word {
	t.Helper()
	now := time.Now().UTC()
	w := model.Word{
		ID:           store.NewID("wd"),
		Word:         "ephemeral",
		Phonetic:     "/ɪˈfem.ər.əl/",
		Definitions:  []model.Definition{{Pos: "adj", Meaning: "short-lived"}},
		Examples:     []string{"Fame is ephemeral."},
		Synonyms:     []string{},
		Advanced:     []string{},
		Inflections:  []model.Inflection{},
		Tags:         []string{"gre"},
		NextReviewAt: now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := wordStore.Add(w); err != nil {
		t.Fatalf("add test word: %v", err)
	}
	return w
}

func addTestTag(t *testing.T, name string) {
	t.Helper()
	if err := tagStore.Add(model.Tag{ID: store.NewID("tag"), Name: name, Color: "#888888"}); err != nil {
		t.Fatalf("add test tag %s: %v", name, err)
	}
}

// ---- Tag tests ----

func TestCLITagList(t *testing.T) {
	setupTestDB(t)
	addTestTag(t, "gre")
	addTestTag(t, "toefl")

	out := captureOutput(func() { tagList() })
	if !strings.Contains(out, "gre") || !strings.Contains(out, "toefl") {
		t.Errorf("tag list should contain gre and toefl: %s", out)
	}
}

func TestCLITagListEmpty(t *testing.T) {
	setupTestDB(t)

	out := captureOutput(func() { tagList() })
	if !strings.Contains(out, "No tags") {
		t.Errorf("empty tags should show message: %s", out)
	}
}

func TestCLITagAdd(t *testing.T) {
	setupTestDB(t)

	if err := tagAdd([]string{"gre"}); err != nil {
		t.Fatalf("tagAdd: %v", err)
	}

	tags, _ := tagStore.Load()
	if len(tags) != 1 || tags[0].Name != "gre" {
		t.Errorf("tag not added correctly: %+v", tags)
	}
}

func TestCLITagAddMissingName(t *testing.T) {
	setupTestDB(t)
	if err := tagAdd([]string{}); err == nil {
		t.Error("tagAdd with no name should error")
	}
}

func TestCLITagDelete(t *testing.T) {
	setupTestDB(t)
	addTestTag(t, "temp")

	if err := tagDelete("temp"); err != nil {
		t.Fatalf("tagDelete: %v", err)
	}
	if _, err := tagStore.Get("temp"); err == nil {
		t.Error("tag should be deleted")
	}
}

// ---- Tag batch assign / unassign ----

func TestCLITagBatchAssignAll(t *testing.T) {
	setupTestDB(t)
	addTestTag(t, "gre")
	addTestWord(t)

	if err := batchTagWords("gre", nil, true, "", true); err != nil {
		t.Fatalf("batchTagWords assign: %v", err)
	}

	words, _ := wordStore.Load()
	if len(words) != 1 || !store.HasString(words[0].Tags, "gre") {
		t.Errorf("tag 'gre' not assigned: tags=%v", words[0].Tags)
	}
}

func TestCLITagBatchUnassignAll(t *testing.T) {
	setupTestDB(t)
	addTestTag(t, "gre")
	w := addTestWord(t)
	w.Tags = append(w.Tags, "gre")
	wordStore.Update(w)

	if err := batchTagWords("gre", nil, true, "", false); err != nil {
		t.Fatalf("batchTagWords unassign: %v", err)
	}

	words, _ := wordStore.Load()
	if store.HasString(words[0].Tags, "gre") {
		t.Error("tag 'gre' should be removed")
	}
}

func TestCLITagBatchAssignByID(t *testing.T) {
	setupTestDB(t)
	addTestTag(t, "gre")
	w := addTestWord(t)

	if err := batchTagWords("gre", []string{w.ID}, false, "", true); err != nil {
		t.Fatalf("batchTagWords assign by id: %v", err)
	}

	words, _ := wordStore.Load()
	if !store.HasString(words[0].Tags, "gre") {
		t.Error("tag 'gre' not assigned")
	}
}

func TestCLITagBatchFilterTag(t *testing.T) {
	setupTestDB(t)
	addTestTag(t, "gre")
	addTestTag(t, "toefl")

	now := time.Now().UTC()
	// w1 has no tags — should be skipped by filterTag=toefl.
	w1 := model.Word{
		ID: store.NewID("wd"), Word: "w1", Tags: nil,
		Definitions: []model.Definition{}, Synonyms: []string{}, Advanced: []string{},
		Examples: []string{}, Inflections: []model.Inflection{},
		NextReviewAt: now, CreatedAt: now, UpdatedAt: now,
	}
	// w2 has toefl — filterTag=toefl selects it, then we assign gre.
	w2 := model.Word{
		ID: store.NewID("wd"), Word: "w2", Tags: []string{"toefl"},
		Definitions: []model.Definition{}, Synonyms: []string{}, Advanced: []string{},
		Examples: []string{}, Inflections: []model.Inflection{},
		NextReviewAt: now, CreatedAt: now, UpdatedAt: now,
	}
	wordStore.Add(w1)
	wordStore.Add(w2)

	if err := batchTagWords("gre", nil, true, "toefl", true); err != nil {
		t.Fatalf("batchTagWords filter: %v", err)
	}

	words, _ := wordStore.Load()
	for _, w := range words {
		if w.Word == "w2" && !store.HasString(w.Tags, "gre") {
			t.Error("w2 (matched by filter) should have gre assigned")
		}
		if w.Word == "w1" && store.HasString(w.Tags, "gre") {
			t.Error("w1 (not matching filter) should NOT have gre assigned")
		}
	}
}

// ---- Word tests ----

func TestCLIWordShow(t *testing.T) {
	setupTestDB(t)
	w := addTestWord(t)

	out := captureOutput(func() { wordShow(w.ID) })
	if !strings.Contains(out, "ephemeral") {
		t.Errorf("word show should contain word text: %s", out)
	}
	if !strings.Contains(out, "short-lived") {
		t.Errorf("word show should contain definition: %s", out)
	}
}

func TestCLIWordShowNotFound(t *testing.T) {
	setupTestDB(t)

	err := wordShow("nonexistent")
	if err == nil {
		t.Error("wordShow with bad id should return error")
	}
}

// ---- Review stats ----

func TestCLIReviewStatsEmpty(t *testing.T) {
	setupTestDB(t)

	out := captureOutput(func() { reviewStats() })
	if !strings.Contains(out, "Review Stats") {
		t.Errorf("review stats should show header: %s", out)
	}
}

// ---- Search ----

func TestCLIRunSearch(t *testing.T) {
	setupTestDB(t)
	addTestWord(t)

	out := captureOutput(func() {
		runSearch([]string{"ephemeral"})
	})
	if !strings.Contains(out, "ephemeral") {
		t.Errorf("search output should contain 'ephemeral': %s", out)
	}
}

func TestCLIRunSearchNoResults(t *testing.T) {
	setupTestDB(t)

	out := captureOutput(func() {
		runSearch([]string{"nonexistent12345"})
	})
	if !strings.Contains(out, "0 results") {
		t.Errorf("search should say '0 results': %s", out)
	}
}

// ---- Command routing ----

func TestRunTag(t *testing.T) {
	setupTestDB(t)
	addTestTag(t, "gre")

	err := runTag([]string{"list"})
	if err != nil {
		t.Errorf("runTag list: %v", err)
	}
}

func TestRunReviewStats(t *testing.T) {
	setupTestDB(t)

	err := runReview([]string{})
	if err != nil {
		t.Errorf("runReview stats: %v", err)
	}
}

func TestRunReviewStartEmpty(t *testing.T) {
	setupTestDB(t)

	err := runReview([]string{"start"})
	if err != nil {
		t.Errorf("runReview start (empty): %v", err)
	}
}

func TestRunReviewInvalid(t *testing.T) {
	setupTestDB(t)

	// Invalid rating input causes a skipped card but no error (returns nil).
	// We can't easily test stdin interaction, but we test the routing.
	err := runReview([]string{"stats"})
	if err != nil {
		t.Errorf("runReview stats: %v", err)
	}
}
