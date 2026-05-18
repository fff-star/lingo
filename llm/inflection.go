package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"lingo/model"
)

// PosMask flags for word categories.
const (
	PosVerb = 1 << iota
	PosNoun
	PosAdj
	PosAdv
)

var posLabel = map[int]string{
	PosVerb: "verb",
	PosNoun: "noun",
	PosAdj:  "adjective",
	PosAdv:  "adverb",
}

// formSpec maps each POS to its expected inflection form labels.
var formSpec = map[int][]string{
	PosVerb: {"past tense", "past participle", "present participle", "3rd person singular"},
	PosNoun: {"plural"},
	PosAdj:  {"comparative", "superlative", "adverb"},
	PosAdv:  {"comparative", "superlative"},
}

// SuggestInflections asks the LLM to suggest inflected forms for a word,
// restricted to the types implied by posMask.
func SuggestInflections(cfg *Config, word string, posMask int) ([]model.Inflection, error) {
	prompt := buildPrompt(word, posMask)

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	resp, err := ChatCompletion(cfg, messages)
	if err != nil {
		return nil, err
	}

	jsonStr := stripMarkdownFences(resp)

	var inflections []model.Inflection
	if err := json.Unmarshal([]byte(jsonStr), &inflections); err != nil {
		return nil, fmt.Errorf("parse inflection response: %w\n\nRaw:\n%s", err, resp)
	}

	return inflections, nil
}

func buildPrompt(word string, posMask int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("List the inflected forms of the English word %q.\n\n", word))

	if posMask&PosVerb != 0 {
		b.WriteString("- Verb forms: past tense, past participle, present participle (-ing), 3rd person singular (-s)\n")
	}
	if posMask&PosNoun != 0 {
		b.WriteString("- Noun form: plural\n")
	}
	if posMask&PosAdj != 0 {
		b.WriteString("- Adjective forms: comparative (-er/more), superlative (-est/most), adverb (-ly)\n")
	}
	if posMask&PosAdv != 0 {
		b.WriteString("- Adverb forms: comparative (-er/more), superlative (-est/most)\n")
	}

	b.WriteString("\nReturn ONLY a JSON array, no markdown fences, no extra text:\n")
	b.WriteString("[\n")

	// Build example JSON entries matching only requested forms.
	var examples []string
	seen := map[string]bool{}
	for mask, labels := range formSpec {
		if posMask&mask == 0 {
			continue
		}
		for _, label := range labels {
			if seen[label] {
				continue
			}
			seen[label] = true
			examples = append(examples, fmt.Sprintf(`  {"form": %q, "value": "..."}`, label))
		}
	}
	b.WriteString(strings.Join(examples, ",\n"))
	b.WriteString("\n]\n\n")

	b.WriteString("If the word has no inflections (e.g., mass noun, uninflected adverb), return [].\n")
	b.WriteString("If the word is ALREADY an inflected form (e.g., \"ran\"), return the other forms relative to the lemma. Do NOT repeat the input word itself.\n")

	return b.String()
}

// NormalizeInflections deduplicates and removes empty values.
func NormalizeInflections(in []model.Inflection) []model.Inflection {
	seen := make(map[string]bool)
	var out []model.Inflection
	for _, i := range in {
		v := strings.TrimSpace(i.Value)
		if v == "" {
			continue
		}
		key := i.Form + ":" + v
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, model.Inflection{Form: i.Form, Value: v})
	}
	return out
}

// PosMaskFromDefs derives a POS mask from a set of definition POS tags.
func PosMaskFromDefs(defs []model.Definition) int {
	mask := 0
	for _, d := range defs {
		pos := strings.ToLower(d.Pos)
		switch {
		case pos == "verb" || strings.HasPrefix(pos, "verb"):
			mask |= PosVerb
		case pos == "noun" || strings.HasPrefix(pos, "noun"):
			mask |= PosNoun
		case pos == "adjective" || strings.HasPrefix(pos, "adj"):
			mask |= PosAdj
		case pos == "adverb" || strings.HasPrefix(pos, "adv"):
			mask |= PosAdv
		}
	}
	return mask
}
