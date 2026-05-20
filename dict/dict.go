package dict

import (
	"lingo/model"
)

// WordInfo is the result of a dictionary lookup.
type WordInfo struct {
	Word           string
	Phonetic       string
	AudioURL       string
	Definitions    []model.Definition // MW English definitions
	ECDefinitions  []model.Definition // ECDICT Chinese definitions
	ECTag          string             // ECDICT edu tags: cet4, toefl, gre, etc.
	Examples       []string
	Inflections    []model.Inflection
}

// Lookup queries Merriam-Webster and ECDICT for a word.
// MW phonetic/audio takes priority; ECDICT phonetic is fallback.
// Requires MW_API_KEY environment variable (MW) and ECDICT_DB_PATH (ECDICT).
func Lookup(word string) (*WordInfo, error) {
	info := &WordInfo{Word: word}

	// 1. MW lookup (English definitions, phonetic, audio).
	raw, mwErr := LookupMW(word)
	if mwErr == nil {
		info.Word = raw.Headword
		if len(raw.Prons) > 0 {
			info.Phonetic = "/" + raw.Prons[0] + "/"
		}
		if raw.AudioFile != "" {
			info.AudioURL = mwAudioURL(raw.AudioFile)
		}
		for _, d := range raw.ShortDefs {
			info.Definitions = append(info.Definitions, model.Definition{
				Pos:     raw.Functional,
				Meaning: d,
			})
		}
		info.Inflections = raw.Inflections
	}

	// 2. ECDICT lookup (Chinese definitions, best-effort).
	if ec, ecErr := LookupECDICT(word); ecErr == nil {
		info.ECDefinitions = ec.Definitions
		info.ECTag = ec.Tag
		if info.Phonetic == "" && ec.Phonetic != "" {
			info.Phonetic = ec.Phonetic
		}
	}

	// 3. Must have at least some data.
	if mwErr != nil && len(info.ECDefinitions) == 0 {
		return nil, mwErr
	}
	if info.Word == "" {
		info.Word = word
	}
	return info, nil
}
