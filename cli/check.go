package cli

import (
	"fmt"
	"strings"
)

func init() {
	register("check", runCheck)
}

func runCheck(args []string) error {
	fmt.Println("═══ Data Integrity Check ═══")
	fmt.Println()

	words, err := wordStore.Load()
	if err != nil {
		return fmt.Errorf("load words: %w", err)
	}
	phrases, err := phraseStore.Load()
	if err != nil {
		return fmt.Errorf("load phrases: %w", err)
	}
	sentences, err := sentenceStore.Load()
	if err != nil {
		return fmt.Errorf("load sentences: %w", err)
	}
	articles, err := articleStore.Load()
	if err != nil {
		return fmt.Errorf("load articles: %w", err)
	}
	tags, err := tagStore.Load()
	if err != nil {
		return fmt.Errorf("load tags: %w", err)
	}

	tagNames := make(map[string]bool)
	for _, t := range tags {
		tagNames[t.Name] = true
	}

	issues := 0

	// Check words.
	wordIDs := make(map[string]bool)
	for _, w := range words {
		if w.Word == "" {
			fmt.Printf("[WARN] word %s has empty word field\n", w.ID)
			issues++
		}
		if wordIDs[w.ID] {
			fmt.Printf("[WARN] duplicate word ID: %s\n", w.ID)
			issues++
		}
		wordIDs[w.ID] = true

		for _, t := range w.Tags {
			if !tagNames[t] {
				fmt.Printf("[WARN] word '%s' uses undefined tag '%s'\n", w.Word, t)
				issues++
			}
		}
	}

	// Check phrases.
	for _, p := range phrases {
		if p.Phrase == "" {
			fmt.Printf("[WARN] phrase %s has empty phrase field\n", p.ID)
			issues++
		}
		for _, t := range p.Tags {
			if !tagNames[t] {
				fmt.Printf("[WARN] phrase '%s' uses undefined tag '%s'\n", p.Phrase, t)
				issues++
			}
		}
	}

	// Check sentences.
	for _, s := range sentences {
		if s.Text == "" {
			fmt.Printf("[WARN] sentence %s has empty text\n", s.ID)
			issues++
		}
		for _, t := range s.Tags {
			if !tagNames[t] {
				fmt.Printf("[WARN] sentence %s uses undefined tag '%s'\n", s.ID, t)
				issues++
			}
		}
	}

	// Check articles.
	for _, a := range articles {
		if a.Title == "" {
			fmt.Printf("[WARN] article %s has empty title\n", a.ID)
			issues++
		}
		for _, t := range a.Tags {
			if !tagNames[t] {
				fmt.Printf("[WARN] article '%s' uses undefined tag '%s'\n", a.Title, t)
				issues++
			}
		}
	}

	// Check for duplicate words.
	seen := make(map[string]string)
	for _, w := range words {
		lower := strings.ToLower(w.Word)
		if exID, ok := seen[lower]; ok {
			fmt.Printf("[WARN] duplicate word '%s' (%s and %s)\n", w.Word, exID, w.ID)
			issues++
		}
		seen[lower] = w.ID
	}

	// Check for duplicate phrases.
	seenPh := make(map[string]string)
	for _, p := range phrases {
		lower := strings.ToLower(p.Phrase)
		if exID, ok := seenPh[lower]; ok {
			fmt.Printf("[WARN] duplicate phrase '%s' (%s and %s)\n", p.Phrase, exID, p.ID)
			issues++
		}
		seenPh[lower] = p.ID
	}

	fmt.Println()
	fmt.Printf("Analyzed: %d words, %d phrases, %d sentences, %d articles, %d tags\n",
		len(words), len(phrases), len(sentences), len(articles), len(tags))

	if issues == 0 {
		fmt.Println("No issues found.")
	} else {
		fmt.Printf("%d issue(s) found.\n", issues)
	}
	return nil
}
