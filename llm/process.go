package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"lingo/model"
)

const maxInputChars = 30000

// ExtractedItems holds all items extracted by the LLM from an article.
type ExtractedItems struct {
	Summary        string              `json:"summary"`
	SuggestedTags  []string            `json:"suggested_tags"`
	Words          []ExtractedWord     `json:"words"`
	Phrases        []ExtractedPhrase   `json:"phrases"`
	Sentences      []ExtractedSentence `json:"sentences"`
	GrammarErrors  []ExtractedGrammar  `json:"grammar_errors"`
	ModelEssay     string              `json:"model_essay"`
	ModelEssay2    *ModelEssay2Result  `json:"model_essay_2,omitempty"`
}

// ModelEssay2Result is an independent model essay on the same topic with its own analysis.
type ModelEssay2Result struct {
	Essay     string              `json:"essay"`
	Words     []ExtractedWord     `json:"words"`
	Phrases   []ExtractedPhrase   `json:"phrases"`
	Sentences []ExtractedSentence `json:"sentences"`
}

// ExtractedGrammar is a grammar error found by LLM analysis.
type ExtractedGrammar struct {
	Sentence    string `json:"sentence"`
	Correction  string `json:"correction"`
	Explanation string `json:"explanation"`
	ErrorType   string `json:"error_type"`
}

// ExtractedWord is a word the LLM identified as worth learning.
type ExtractedWord struct {
	Word        string            `json:"word"`
	Definitions []model.Definition `json:"definitions"`
	Example     string            `json:"example"`
	Synonyms    []string          `json:"synonyms"`
	Notes       string            `json:"notes"`
}

// ExtractedPhrase is a phrase the LLM identified as worth learning.
type ExtractedPhrase struct {
	Phrase     string   `json:"phrase"`
	Type       string   `json:"type"`
	Words      []string `json:"words"`
	Definition string   `json:"definition"`
	Example    string   `json:"example"`
	Synonyms   []string `json:"synonyms"`
	Notes      string   `json:"notes"`
}

// ExtractedSentence is a sentence the LLM identified as worth saving.
type ExtractedSentence struct {
	Text           string   `json:"text"`
	Translation    string   `json:"translation"`
	Why            string   `json:"why"`
	SuggestedTags  []string `json:"suggested_tags"`
}

// ----- prompt -----

const systemPrompt = `You are a language-learning material curator. Given an English article, extract vocabulary, phrases, and sentences worth saving for a non-native English learner (Chinese speaker, B2–C1 level). Output ONLY valid JSON — no markdown fences, no extra text.

## Word Selection

Pick words that meet at least one of:
- Academic / advanced vocabulary (GRE, TOEFL, IELTS tier)
- Domain-specific or technical terms central to the article's topic
- Precise verbs, adjectives, or adverbs that elevate writing quality
- Abstract nouns that express complex ideas concisely

Skip: basic function words, common A1–B1 words, pronouns, articles, everyday concrete nouns (table, book, walk).

For each word:
- "word": the headword (lowercase unless proper noun)
- "definitions": array of { "pos": "n./adj./v./adv.", "meaning": "definition IN CHINESE" }
- "example": ONE original sentence from the article containing this word
- "synonyms": 1–3 near-synonyms in English (can be empty array)
- "notes": usage tip or collocation note IN CHINESE (can be empty string)

Limit: 20–40 words. If fewer qualify, return fewer; do not pad with weak choices.

## Phrase Selection

Extract fixed expressions, not random word combinations:
- Phrasal verbs (e.g., "carry out", "put up with")
- Idioms (e.g., "in the long run", "break the ice")
- Strong collocations (e.g., "heavy rain", "make a decision", "pose a threat")
- Discourse connectives / transitional phrases (e.g., "on the other hand", "that being said")

For each phrase:
- "phrase": the full phrase
- "type": one of "phrasal_verb" | "idiom" | "collocation" | "connective" | "other"
- "words": array of individual words in the phrase
- "definition": meaning IN CHINESE
- "example": ONE original sentence from the article
- "synonyms": 1–3 alternative expressions (can be empty array)
- "notes": usage note IN CHINESE (can be empty string)

Limit: 3–10 phrases. If none qualify, return empty array.

## Sentence Selection

Select sentences that are worth saving as writing/speaking models:
- Well-structured sentences with reusable patterns
- Sentences that express a compelling or nuanced idea
- Opening/closing sentences of arguments
- Sentences containing advanced grammar worth studying

Skip: simple factual statements, transitional one-liners, dialogue fragments.

For each sentence:
- "text": the exact sentence text
- "translation": natural Chinese translation
- "why": why this sentence is worth learning, IN CHINESE (be specific: what pattern, what usage)
- "suggested_tags": 1–3 tags like "argument", "writing", "society", "technology", etc.

Limit: 3–8 sentences.

## Summary & Tags

- "summary": 2–4 sentence summary of the article IN CHINESE
- "suggested_tags": 1–5 tags for the article itself (lowercase English), e.g. ["technology", "ethics", "argument"]

## Output JSON Structure

{
  "summary": "...",
  "suggested_tags": ["...", "..."],
  "words": [
    {
      "word": "...",
      "definitions": [{"pos": "adj.", "meaning": "中文释义"}],
      "example": "...",
      "synonyms": ["...", "..."],
      "notes": "中文使用说明"
    }
  ],
  "phrases": [
    {
      "phrase": "...",
      "type": "idiom",
      "words": ["...", "..."],
      "definition": "中文释义",
      "example": "...",
      "synonyms": ["...", "..."],
      "notes": "中文使用说明"
    }
  ],
  "sentences": [
    {
      "text": "...",
      "translation": "中文翻译",
      "why": "为什么值得保存（中文）",
      "suggested_tags": ["...", "..."]
    }
  ]
}`

// ----- main API -----

// ProcessArticle sends article content to the LLM and returns extracted items.
func ProcessArticle(cfg *Config, content, title string) (*ExtractedItems, error) {
	input := truncateRunes(content, maxInputChars)

	userMsg := fmt.Sprintf("Title: %s\n\nContent:\n%s", title, input)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}

	resp, err := ChatCompletion(cfg, messages)
	if err != nil {
		return nil, err
	}

	return parseResponse(resp)
}

// truncateRunes trims s to at most n runes, cutting at a word boundary.
func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	cut := string(runes[:n])
	if idx := strings.LastIndexByte(cut, ' '); idx > 0 {
		cut = cut[:idx]
	}
	return cut
}

// ToAIAnalysis converts extracted items to a model.AIAnalysis for inline storage.
func (e *ExtractedItems) ToAIAnalysis() *model.AIAnalysis {
	analysis := &model.AIAnalysis{
		Summary:       e.Summary,
		SuggestedTags: e.SuggestedTags,
	}
	for _, w := range e.Words {
		analysis.Words = append(analysis.Words, model.ExtractedWord{
			Word:        w.Word,
			Definitions: w.Definitions,
			Example:     w.Example,
			Synonyms:    w.Synonyms,
			Notes:       w.Notes,
		})
	}
	for _, p := range e.Phrases {
		analysis.Phrases = append(analysis.Phrases, model.ExtractedPhrase{
			Phrase:     p.Phrase,
			Type:       p.Type,
			Words:      p.Words,
			Definition: p.Definition,
			Example:    p.Example,
			Synonyms:   p.Synonyms,
			Notes:      p.Notes,
		})
	}
	for _, s := range e.Sentences {
		analysis.Sentences = append(analysis.Sentences, model.ExtractedSentence{
			Text:          s.Text,
			Translation:   s.Translation,
			Why:           s.Why,
			SuggestedTags: s.SuggestedTags,
		})
	}
	analysis.ModelEssay = e.ModelEssay
	if e.ModelEssay2 != nil {
		m2 := &model.ModelEssay2{
			Essay: e.ModelEssay2.Essay,
		}
		for _, w := range e.ModelEssay2.Words {
			m2.Words = append(m2.Words, model.ExtractedWord{
				Word: w.Word, Definitions: w.Definitions,
				Example: w.Example, Synonyms: w.Synonyms, Notes: w.Notes,
			})
		}
		for _, p := range e.ModelEssay2.Phrases {
			m2.Phrases = append(m2.Phrases, model.ExtractedPhrase{
				Phrase: p.Phrase, Type: p.Type, Words: p.Words,
				Definition: p.Definition, Example: p.Example,
				Synonyms: p.Synonyms, Notes: p.Notes,
			})
		}
		for _, s := range e.ModelEssay2.Sentences {
			m2.Sentences = append(m2.Sentences, model.ExtractedSentence{
				Text: s.Text, Translation: s.Translation,
				Why: s.Why, SuggestedTags: s.SuggestedTags,
			})
		}
		analysis.ModelEssay2 = m2
	}
	for _, g := range e.GrammarErrors {
		analysis.GrammarErrors = append(analysis.GrammarErrors, model.GrammarError{
			Sentence:    g.Sentence,
			Correction:  g.Correction,
			Explanation: g.Explanation,
			ErrorType:   g.ErrorType,
		})
	}
	return analysis
}

// ----- parsing -----

func parseResponse(raw string) (*ExtractedItems, error) {
	jsonStr := stripMarkdownFences(raw)

	var items ExtractedItems
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w\n\nRaw response:\n%s", err, raw)
	}

	return &items, nil
}

// ----- composition analysis -----

const compositionSystemPrompt = `You are an English writing coach. Analyze the following student composition (non-native English learner, Chinese speaker, B2–C1 level). Output ONLY valid JSON — no markdown fences, no extra text.

## Analysis Goals

1. Provide a brief overall assessment (2-3 sentences IN CHINESE)
2. Identify vocabulary the student used well or could improve
3. Identify phrases/expressions used or opportunities for better expressions
4. Identify well-structured sentences worth keeping as writing models
5. Find grammar, wording, and collocation errors that need correction
6. Suggest tags for the composition

## Grammar Error Detection

Find errors in grammar, word choice, collocation, or sentence structure. Be specific and helpful.

For each error:
- "sentence": the original sentence containing the error
- "correction": the corrected sentence (fix only the error, keep the rest unchanged)
- "explanation": what the error is and why it's wrong, IN CHINESE
- "error_type": one of "grammar" | "word_choice" | "collocation" | "tense" | "preposition" | "article" | "word_order" | "other"

Only flag errors that clearly impact correctness or naturalness. Do NOT flag stylistic preferences or subjective improvements.

## Model Essay (范文)

Write a polished model version of the student's composition. Requirements:
- Fix all grammar, wording, and collocation errors you identified
- Preserve the student's original ideas, arguments, and structure
- Elevate vocabulary and expressions where appropriate (use the words/phrases you suggested)
- Improve sentence variety and flow
- Keep the same approximate length (do NOT expand into a much longer essay)
- Output as "model_essay": a single string with the full polished essay

This is NOT a summary or commentary — it is the complete rewritten essay that the student can study as a reference.

## Model Essay 2 (Independent — same topic, fresh writing)

Write a SECOND model essay on the Topic provided by the user, completely independent of the student's composition. Requirements:
- Do NOT reference, follow, or be constrained by the student's original essay
- Write a fresh, well-structured English essay on the TOPIC from your own perspective
- Cover the general themes of the topic naturally
- Keep the length similar to the student's composition
- This is an exemplar of good English writing on this topic for the student to study
- If the user did not provide a topic, infer a general topic from the composition title and content

After writing this essay, extract words, phrases, and sentences from it following the same rules as the student-facing analysis. Each extracted item MUST include a detailed explanation IN CHINESE explaining why the word/phrase/sentence is worth learning — its nuance, usage context, or grammatical interest.

Output as "model_essay_2": an object with:
- "essay": the full essay text
- "words": 5–10 words from this essay, each with Chinese explanations in "notes"
- "phrases": 3–8 phrases from this essay, each with Chinese explanations in "notes"
- "sentences": 2–5 sentences from this essay, each with Chinese "translation" and "why"

## Word Selection

Pick words that are:
- Used effectively and worth noting for future use
- Could be replaced with more precise/advanced alternatives

For each word:
- "word": the headword
- "definitions": array of { "pos": "n./adj./v./adv.", "meaning": "definition IN CHINESE" }
- "example": the sentence from the composition containing this word
- "synonyms": 1–3 alternative words (can be empty array)
- "notes": usage tip or improvement suggestion IN CHINESE (can be empty string)

Limit: 10–15 words. If fewer qualify, return fewer; do not pad with weak choices.

## Phrase Selection

Identify phrases the student used or could use:
- Collocations and fixed expressions
- Phrasal verbs, idioms, discourse connectives

For each phrase:
- "phrase": the full phrase
- "type": one of "phrasal_verb" | "idiom" | "collocation" | "connective" | "other"
- "words": array of individual words in the phrase
- "definition": meaning IN CHINESE
- "example": the sentence from the composition
- "synonyms": 1–3 alternative expressions (can be empty array)
- "notes": usage note IN CHINESE (can be empty string)

Limit: 2–8 phrases.

## Sentence Selection

Select sentences from the composition that:
- Are well-structured and could serve as writing models
- Contain interesting grammar or expressions
- Express ideas particularly effectively

For each sentence:
- "text": the exact sentence text
- "translation": natural Chinese translation
- "why": why this sentence is noteworthy, IN CHINESE
- "suggested_tags": 1–3 relevant tags

Limit: 2–6 sentences.

## Output JSON Structure

{
  "summary": "总体评价（中文，2-3句）",
  "suggested_tags": ["tag1", "tag2"],
  "words": [
    {
      "word": "...",
      "definitions": [{"pos": "adj.", "meaning": "中文释义"}],
      "example": "...",
      "synonyms": ["...", "..."],
      "notes": "中文使用建议"
    }
  ],
  "phrases": [
    {
      "phrase": "...",
      "type": "idiom",
      "words": ["...", "..."],
      "definition": "中文释义",
      "example": "...",
      "synonyms": ["...", "..."],
      "notes": "中文使用建议"
    }
  ],
  "sentences": [
    {
      "text": "...",
      "translation": "中文翻译",
      "why": "为什么值得关注（中文）",
      "suggested_tags": ["...", "..."]
    }
  ],
  "grammar_errors": [
    {
      "sentence": "原句",
      "correction": "修正后的句子",
      "explanation": "中文解释：为什么是错误、如何改正",
      "error_type": "grammar"
    }
  ],
  "model_essay": "完整的范文（英文）",
  "model_essay_2": {
    "essay": "独立的范文（英文，不参考学生原文）",
    "words": [
      {
        "word": "...",
        "definitions": [{"pos": "adj.", "meaning": "中文释义"}],
        "example": "...",
        "synonyms": ["...", "..."],
        "notes": "中文讲解：这个词的用法、语境、值得学习的原因"
      }
    ],
    "phrases": [
      {
        "phrase": "...",
        "type": "idiom",
        "words": ["...", "..."],
        "definition": "中文释义",
        "example": "...",
        "synonyms": ["...", "..."],
        "notes": "中文讲解：这个短语的用法和语境"
      }
    ],
    "sentences": [
      {
        "text": "...",
        "translation": "中文翻译",
        "why": "为什么这个句子值得学习（中文）",
        "suggested_tags": ["...", "..."]
      }
    ]
  }
}`

// AnalyzeComposition analyzes a student's composition and returns feedback.
// Unlike ProcessArticle, results are meant to be displayed inline, not added to stores.
func AnalyzeComposition(cfg *Config, content, title, topic string) (*ExtractedItems, error) {
	input := truncateRunes(content, maxInputChars)

	userMsg := fmt.Sprintf("Title: %s\nTopic: %s\n\nComposition:\n%s", title, topic, input)

	messages := []Message{
		{Role: "system", Content: compositionSystemPrompt},
		{Role: "user", Content: userMsg},
	}

	resp, err := ChatCompletion(cfg, messages)
	if err != nil {
		return nil, err
	}

	return parseResponse(resp)
}

func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)

	// Remove ```json ... ``` wrapper.
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}

	return s
}
