package dict

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"lingo/model"
)

const mwBaseURL = "https://www.dictionaryapi.com/api/v3/references/collegiate/json/"

// MWResult holds the parsed inflection data from Merriam-Webster.
type MWResult struct {
	Stems       []string
	Inflections []model.Inflection
	Headword    string
	Functional  string   // part of speech
	ShortDefs   []string // short definitions (max 3)
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

	// Collect from all homograph entries.
	for _, e := range entries {
		if result.Headword == "" {
			result.Headword = e.Hwi.Hw
		}
		if result.Functional == "" && e.Fl != "" {
			result.Functional = e.Fl
		}
		result.Stems = append(result.Stems, e.Meta.Stems...)
		result.ShortDefs = append(result.ShortDefs, e.ShortDef...)

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
			if value != "" {
				result.Inflections = append(result.Inflections, model.Inflection{
					Form:  label,
					Value: value,
				})
			}
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
