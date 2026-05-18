package dict

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lingo/model"
)

const apiURL = "https://api.dictionaryapi.dev/api/v2/entries/en/"

// WordInfo is the result of a dictionary lookup.
type WordInfo struct {
	Word        string
	Phonetic    string
	Definitions []model.Definition
	Examples    []string
	Inflections []model.Inflection
}

// Lookup queries the Free Dictionary API for a word.
func Lookup(word string) (*WordInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL + word)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("word not found")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var entries []dictEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("word not found")
	}

	info := &WordInfo{Word: word}

	for _, entry := range entries {
		if info.Phonetic == "" && entry.Phonetic != "" {
			info.Phonetic = entry.Phonetic
		}
		// Also check phonetic in nested phonetics array.
		for _, p := range entry.Phonetics {
			if info.Phonetic == "" && p.Text != "" {
				info.Phonetic = p.Text
			}
		}

		for _, m := range entry.Meanings {
			for _, d := range m.Definitions {
				def := model.Definition{Pos: m.PartOfSpeech, Meaning: d.Definition}
				info.Definitions = append(info.Definitions, def)
				if d.Example != "" {
					info.Examples = append(info.Examples, d.Example)
				}
				// Collect synonyms from the meaning level.
				for _, s := range m.Synonyms {
					info.Inflections = appendIfNew(info.Inflections, model.Inflection{Form: "synonym", Value: s})
				}
			}
		}
	}

	// Limit examples to 5.
	if len(info.Examples) > 5 {
		info.Examples = info.Examples[:5]
	}

	if info.Phonetic != "" && !strings.HasPrefix(info.Phonetic, "/") {
		info.Phonetic = "/" + info.Phonetic + "/"
	}

	return info, nil
}

type dictEntry struct {
	Word      string       `json:"word"`
	Phonetic  string       `json:"phonetic"`
	Phonetics []phonetic   `json:"phonetics"`
	Meanings  []meaning    `json:"meanings"`
}

type phonetic struct {
	Text string `json:"text"`
}

type meaning struct {
	PartOfSpeech string     `json:"partOfSpeech"`
	Synonyms     []string   `json:"synonyms"`
	Definitions  []definition `json:"definitions"`
}

type definition struct {
	Definition string `json:"definition"`
	Example    string `json:"example"`
}

func appendIfNew(items []model.Inflection, item model.Inflection) []model.Inflection {
	for _, ex := range items {
		if ex.Value == item.Value {
			return items
		}
	}
	return append(items, item)
}

// VerbFormTypes lists inflection types that require verb definitions.
var verbFormTypes = map[string]bool{
	"past tense":          true,
	"past participle":     true,
	"present participle":  true,
	"3rd person singular": true,
}

// VerifyInflection checks that a candidate inflected form is semantically consistent
// with the claimed inflection type by looking it up and checking part-of-speech.
// Returns true only if the API confirms the word AND its POS matches the form type.
func VerifyInflection(baseWord, formValue, formType string) bool {
	info, err := Lookup(formValue)
	if err != nil {
		return false
	}

	if verbFormTypes[formType] {
		return hasPOS(info, "verb")
	}
	if formType == "plural" {
		return hasPOS(info, "noun")
	}
	if formType == "comparative" || formType == "superlative" {
		return hasPOS(info, "adjective") || hasPOS(info, "adverb")
	}

		if formType == "adverb" {
		return hasPOS(info, "adverb")
		}

	return true
}

func hasPOS(info *WordInfo, target string) bool {
	for _, d := range info.Definitions {
		if d.Pos == target {
			return true
		}
	}
	return false
}
