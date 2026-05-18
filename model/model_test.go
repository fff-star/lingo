package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWordJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC()
	w := Word{
		ID:          "wd_test123",
		Word:        "ephemeral",
		Phonetic:    "/ɪˈfem.ər.əl/",
		Definitions: []Definition{{Pos: "adj", Meaning: "short-lived"}},
		Examples:    []string{"Life is brief."},
		Inflections: []Inflection{{Form: "noun", Value: "ephemerality"}},
		Synonyms:    []string{"transient"},
		Advanced:    []string{"evanescent"},
		Tags:        []string{"gre"},
		Notes:       "test note",
		Stability:   3.5,
		Difficulty:  5.0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	b, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Word
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != w.ID {
		t.Errorf("ID = %q, want %q", got.ID, w.ID)
	}
	if got.Word != w.Word {
		t.Errorf("Word = %q, want %q", got.Word, w.Word)
	}
	if got.Phonetic != w.Phonetic {
		t.Errorf("Phonetic = %q, want %q", got.Phonetic, w.Phonetic)
	}
	if len(got.Definitions) != 1 {
		t.Errorf("len(Definitions) = %d, want 1", len(got.Definitions))
	}
	if got.Definitions[0].Pos != "adj" || got.Definitions[0].Meaning != "short-lived" {
		t.Errorf("Definition mismatch")
	}
	if got.Stability != 3.5 {
		t.Errorf("Stability = %f, want 3.5", got.Stability)
	}
	if got.Difficulty != 5.0 {
		t.Errorf("Difficulty = %f, want 5.0", got.Difficulty)
	}
}

func TestPhraseJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC()
	p := Phrase{
		ID:         "ph_test456",
		Phrase:     "in the long run",
		Type:       "idiom",
		Words:      []string{"in", "the", "long", "run"},
		Definition: "eventually",
		Examples:   []string{"It pays off in the long run."},
		Synonyms:   []string{"in the end"},
		Advanced:   []string{"ultimately"},
		Tags:       []string{"writing"},
		Stability:  0,
		Difficulty: 5.0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Phrase
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Phrase != p.Phrase {
		t.Errorf("Phrase = %q, want %q", got.Phrase, p.Phrase)
	}
	if got.Type != "idiom" {
		t.Errorf("Type = %q, want idiom", got.Type)
	}
	if len(got.Words) != 4 {
		t.Errorf("len(Words) = %d, want 4", len(got.Words))
	}
}

func TestDefinitionJSONRoundTrip(t *testing.T) {
	d := Definition{Pos: "verb", Meaning: "to run fast"}
	b, _ := json.Marshal(d)
	var got Definition
	json.Unmarshal(b, &got)
	if got.Pos != "verb" || got.Meaning != "to run fast" {
		t.Error("Definition round-trip failed")
	}
}

func TestInflectionJSONRoundTrip(t *testing.T) {
	i := Inflection{Form: "past tense", Value: "ran"}
	b, _ := json.Marshal(i)
	var got Inflection
	json.Unmarshal(b, &got)
	if got.Form != "past tense" || got.Value != "ran" {
		t.Error("Inflection round-trip failed")
	}
}

func TestWordEmptyArrays(t *testing.T) {
	// Verify that json.Marshal outputs [] not null for empty arrays.
	w := Word{ID: "wd_x", Word: "test", Definitions: []Definition{}, Synonyms: []string{}, Advanced: []string{}, Examples: []string{}, Inflections: []Inflection{}, Tags: []string{}}
	b, _ := json.Marshal(w)
	s := string(b)
	if s == "" {
		t.Fatal("empty marshal")
	}
	// Go's json encoder marshals nil slices as "null", but empty initialized slices
	// should marshal as "[]". Since we explicitly set them to []intial{}, they
	// should appear as [].
	if !contains(s, `"definitions":[]`) && !contains(s, `"definitions": []`) {
		t.Errorf("definitions should be []: %s", s)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
