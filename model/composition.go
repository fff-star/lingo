package model

import "time"

// AIAnalysis holds the results of AI analysis on a composition.
type AIAnalysis struct {
	Summary       string              `json:"summary"`
	Words         []ExtractedWord     `json:"words"`
	Phrases       []ExtractedPhrase   `json:"phrases"`
	Sentences     []ExtractedSentence `json:"sentences"`
	GrammarErrors []GrammarError      `json:"grammar_errors"`
	SuggestedTags []string            `json:"suggested_tags"`
}

// GrammarError is a grammar/wording error found by AI analysis.
type GrammarError struct {
	Sentence    string `json:"sentence"`
	Correction  string `json:"correction"`
	Explanation string `json:"explanation"`
	ErrorType   string `json:"error_type"`
}

// ExtractedWord is a word identified by AI analysis.
type ExtractedWord struct {
	Word        string       `json:"word"`
	Definitions []Definition `json:"definitions"`
	Example     string       `json:"example"`
	Synonyms    []string     `json:"synonyms"`
	Notes       string       `json:"notes"`
}

// ExtractedPhrase is a phrase identified by AI analysis.
type ExtractedPhrase struct {
	Phrase     string   `json:"phrase"`
	Type       string   `json:"type"`
	Words      []string `json:"words"`
	Definition string   `json:"definition"`
	Example    string   `json:"example"`
	Synonyms   []string `json:"synonyms"`
	Notes      string   `json:"notes"`
}

// ExtractedSentence is a sentence identified by AI analysis.
type ExtractedSentence struct {
	Text          string   `json:"text"`
	Translation   string   `json:"translation"`
	Why           string   `json:"why"`
	SuggestedTags []string `json:"suggested_tags"`
}

// Composition is a user-written essay with AI analysis.
type Composition struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Author     string      `json:"author"`
	Content    string      `json:"content"`
	Tags       []string    `json:"tags"`
	Notes      string      `json:"notes"`
	AIAnalysis *AIAnalysis `json:"ai_analysis,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}
