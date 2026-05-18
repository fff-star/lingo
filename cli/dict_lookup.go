package cli

import (
	"lingo/dict"
	"lingo/model"
)

// LookupWord bridges CLI to the dict package.
func LookupWord(word string) (*dict.WordInfo, error) {
	return dict.Lookup(word)
}

// Ensure model is used (required by other files in this package).
var _ = model.Word{}
