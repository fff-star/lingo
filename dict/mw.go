package dict

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"os"
	"time"

	"lingo/model"
)

const mwBaseURL = "https://www.dictionaryapi.com/api/v3/references/collegiate/json/"

// MWResult holds the parsed data from Merriam-Webster.
type MWResult struct {
	Stems       []string
	Inflections []model.Inflection
	Headword    string
	Functional  string   // part of speech label
	Prons       []string // pronunciation strings (cleaned)
	AudioFile   string   // sound filename from prs
	ShortDefs   []string // short definitions
}

// LookupMW queries the Merriam-Webster Collegiate Dictionary for inflections.
// Requires MW_API_KEY environment variable.
func LookupMW(word string) (*MWResult, error) {
	key := os.Getenv("MW_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("MW_API_KEY not set")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(mwBaseURL + word + "?key=" + key)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	// MW returns either []entry or []string (spelling suggestions).
	if len(raw) > 0 && raw[0] == '"' {
		var suggestions []string
		if err := json.Unmarshal(raw, &suggestions); err != nil {
			return nil, fmt.Errorf("parse suggestions: %w", err)
		}
		return nil, fmt.Errorf("word not found, suggestions: %v", suggestions)
	}

	var entries []mwEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("parse entries: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("word not found")
	}

	result := &MWResult{}
	var uncertain []int // indices into result.Inflections needing cxs lookup

	// Collect from all homograph entries.
	for _, e := range entries {
		if result.Headword == "" {
			result.Headword = cleanMWMarkup(e.Hwi.Hw)
		}
		if result.Functional == "" && e.Fl != "" {
			result.Functional = e.Fl
		}
		result.Stems = append(result.Stems, e.Meta.Stems...)
		result.ShortDefs = append(result.ShortDefs, e.ShortDef...)

		for _, pr := range e.Hwi.Prs {
			if pr.Mw != "" {
				result.Prons = append(result.Prons, cleanMWMarkup(pr.Mw))
			}
			if result.AudioFile == "" && pr.Sound.Audio != "" {
				result.AudioFile = pr.Sound.Audio
			}
		}

		for _, infl := range e.Ins {
			label := infl.Il
			if label == "" {
				label = infl.Spl
			}
			// For cutback inflections, use the headword to construct the full form.
			// The "if" field already contains the full form marked with * for syllables.
			value := cleanMWMarkup(infl.If)
			if value == "" && infl.Ifc != "" {
				value = cleanMWMarkup(infl.Ifc)
			}
			// Skip phrasal verb inflections (multi-word values).
			if strings.Contains(value, " ") {
				continue
			}

			if value != "" {
				if label == "" {
					var conf bool
					label, conf = inferInflectionLabel(e.Fl, value)
					if !conf {
						uncertain = append(uncertain, len(result.Inflections))
					}
				} else if label == "or" || label == "also" || strings.HasPrefix(label, "also ") || strings.HasPrefix(label, "or ") {
					// "or"/"also" are connectors, not grammatical labels; copy the previous label.
					if len(result.Inflections) > 0 {
						label = result.Inflections[len(result.Inflections)-1].Form
					}
				}
				result.Inflections = append(result.Inflections, model.Inflection{
					Form:  label,
					Value: value,
				})
			}
		}
	}

	// Resolve uncertain labels via MW cross-reference lookup.
	for _, idx := range uncertain {
		if label := lookupCXSLabel(key, result.Inflections[idx].Value); label != "" {
			result.Inflections[idx].Form = label
		}
	}

	// Deduplicate stems.
	seen := make(map[string]bool)
	var stems []string
	for _, s := range result.Stems {
		if !seen[s] {
			seen[s] = true
			stems = append(stems, s)
		}
	}
	result.Stems = stems

	return result, nil
}

// mwAudioURL constructs the full MP3 URL from an MW audio filename.
func mwAudioURL(audio string) string {
	if audio == "" {
		return ""
	}
	subdir := audio[:1]
	if len(audio) >= 3 {
		if audio[:3] == "bix" {
			subdir = "bix"
		} else if audio[:2] == "gg" {
			subdir = "gg"
		}
	}
	return "https://media.merriam-webster.com/audio/prons/en/us/mp3/" + subdir + "/" + audio + ".mp3"
}

// cleanMWMarkup removes syllable markers (*) from MW strings.
func cleanMWMarkup(s string) string {
	if s == "" {
		return s
	}
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '*' {
			continue
		}
		b = append(b, s[i])
	}
	return string(b)
}

// ----- MW JSON structures -----

type mwEntry struct {
	Meta     mwMeta   `json:"meta"`
	Hwi      mwHwi    `json:"hwi"`
	Fl       string   `json:"fl"`
	Ins      []mwIns  `json:"ins"`
	ShortDef []string `json:"shortdef"`
}

type mwMeta struct {
	ID    string   `json:"id"`
	Stems []string `json:"stems"`
}

type mwHwi struct {
	Hw  string  `json:"hw"`
	Prs []mwPrs `json:"prs"`
}

type mwPrs struct {
	Mw    string   `json:"mw"`
	Sound mwSound  `json:"sound"`
}

type mwSound struct {
	Audio string `json:"audio"`
}

type mwIns struct {
	If  string `json:"if"`  // full inflection form
	Ifc string `json:"ifc"` // cutback form
	Il  string `json:"il"`  // label: "past tense", "plural", "or", "also"
	Spl string `json:"spl"` // sense-specific plural label
}

// inferInflectionLabel guesses an inflection label from the part of speech
// when MW provides neither il nor spl.
// The second return value is false when the label is a fallback guess
// (no suffix matched) and the caller should try MW cross-reference lookup.
func inferInflectionLabel(fl, value string) (label string, confident bool) {
	// Extract first word for phrasal verbs ("cried down" → "cried").
	first := value
	if idx := strings.Index(value, " "); idx >= 0 {
		first = value[:idx]
	}

	// Suffix-based detection, ordered from most specific to general.
	if strings.HasSuffix(first, "ing") {
		return "present participle", true
	}
	if strings.HasSuffix(first, "est") {
		return "superlative", true
	}
	if strings.HasSuffix(first, "er") {
		return "comparative", true
	}
	if strings.HasSuffix(first, "ed") {
		return "past tense", true
	}
	if strings.HasSuffix(first, "en") {
		return "past participle", true
	}
	if strings.HasSuffix(first, "es") || strings.HasSuffix(first, "s") {
		if strings.Contains(fl, "verb") {
			return "3rd person singular", true
		}
		return "plural", true
	}

	// Fallback part-of-speech heuristics for irregular forms.
	if strings.Contains(fl, "verb") {
		return "past tense", false
	}
	if strings.Contains(fl, "noun") {
		return "plural", false
	}
	return "also", false
}

// lookupCXSLabel queries the MW API for an inflected form and extracts
// the cross-reference label (cxl) if present.
func lookupCXSLabel(key, word string) string {
	// Query only the first word for phrasal forms ("ran across" → "ran").
	first := word
	if idx := strings.Index(word, " "); idx >= 0 {
		first = word[:idx]
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(mwBaseURL + first + "?key=" + key)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	var entries []struct {
		CXS []struct {
			CXL   string `json:"cxl"`
			CXTIS []struct {
				CXT string `json:"cxt"`
			} `json:"cxtis"`
		} `json:"cxs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return ""
	}

	for _, e := range entries {
		for _, c := range e.CXS {
			if c.CXL != "" {
				// "past tense of" → "past tense"
				// "past participle of" → "past participle"
				// "past tense and past participle of" → "past tense / past participle"
				return strings.TrimSuffix(c.CXL, " of")
			}
		}
	}
	return ""
}
