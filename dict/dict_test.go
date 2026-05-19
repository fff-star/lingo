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
