package dict

import (
	"fmt"

	"lingo/model"
)

// WordInfo is the result of a dictionary lookup.
type WordInfo struct {
	Word        string
	Phonetic    string
	AudioURL    string
	Definitions []model.Definition
	Examples    []string
	Inflections []model.Inflection
}

// Lookup queries the Merriam-Webster Collegiate Dictionary for a word.
// Requires MW_API_KEY environment variable.
func Lookup(word string) (*WordInfo, error) {
	raw, err := LookupMW(word)
	if err != nil {
		return nil, err
	}

	info := &WordInfo{
		Word:        raw.Headword,
		Inflections: raw.Inflections,
	}

	// Phonetic from MW pronunciation.
	if len(raw.Prons) > 0 {
		info.Phonetic = "/" + raw.Prons[0] + "/"
	}

	// Audio URL from MW sound file.
	if raw.AudioFile != "" {
		info.AudioURL = mwAudioURL(raw.AudioFile)
	}

	// Definitions grouped by part of speech.
	for _, d := range raw.ShortDefs {
		info.Definitions = append(info.Definitions, model.Definition{
			Pos:     raw.Functional,
			Meaning: d,
		})
	}

	if info.Word == "" {
		return nil, fmt.Errorf("word not found")
	}

	return info, nil
}
