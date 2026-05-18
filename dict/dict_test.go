package dict

import (
	"testing"

	"lingo/model"
)

func TestAppendIfNew(t *testing.T) {
	items := []model.Inflection{{Form: "synonym", Value: "fast"}}

	// New item should be appended.
	items = appendIfNew(items, model.Inflection{Form: "synonym", Value: "quick"})
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}

	// Duplicate should be ignored.
	items = appendIfNew(items, model.Inflection{Form: "synonym", Value: "fast"})
	if len(items) != 2 {
		t.Errorf("len after dup = %d, want 2", len(items))
	}
}

func TestHasPOS(t *testing.T) {
	info := &WordInfo{
		Definitions: []model.Definition{
			{Pos: "verb", Meaning: "to move"},
			{Pos: "noun", Meaning: "a movement"},
		},
	}

	if !hasPOS(info, "verb") {
		t.Error("should have verb")
	}
	if !hasPOS(info, "noun") {
		t.Error("should have noun")
	}
	if hasPOS(info, "adjective") {
		t.Error("should not have adjective")
	}
}

func TestVerbFormTypes(t *testing.T) {
	// Ensure all expected verb form types are registered.
	expected := []string{"past tense", "past participle", "present participle", "3rd person singular"}
	for _, ft := range expected {
		if !verbFormTypes[ft] {
			t.Errorf("verbFormTypes missing %q", ft)
		}
	}
}
