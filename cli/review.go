package cli

import (
	"fmt"
	"strings"
	"time"

	"lingo/model"
	"lingo/review"
	"lingo/store"
)

func init() {
	register("review", runReview)
}

func runReview(args []string) error {
	if len(args) == 0 {
		return reviewStats()
	}

	switch args[0] {
	case "start":
		return reviewStart(args[1:])
	case "stats":
		return reviewStats()
	default:
		return reviewStats()
	}
}

func reviewStats() error {
	words, err := wordStore.Load()
	if err != nil {
		return err
	}

	var totalReviewed int
	for _, w := range words {
		if w.ReviewCount > 0 {
			totalReviewed++
		}
	}

	dueWords, err := wordStore.LoadDue()
	if err != nil {
		return err
	}

	phrases, err := phraseStore.Load()
	if err != nil {
		return err
	}

	var totalPhraseReviewed int
	for _, p := range phrases {
		if p.ReviewCount > 0 {
			totalPhraseReviewed++
		}
	}

	duePhrases, err := phraseStore.LoadDue()
	if err != nil {
		return err
	}

	fmt.Println("═══════ Review Stats ═══════")
	fmt.Println()
	fmt.Printf("  Words:   %d due, %d reviewed, %d total\n", len(dueWords), totalReviewed, len(words))
	fmt.Printf("  Phrases: %d due, %d reviewed, %d total\n", len(duePhrases), totalPhraseReviewed, len(phrases))
	fmt.Println()
	totalDue := len(dueWords) + len(duePhrases)
	newWords := len(words) - totalReviewed
	newPhrases := len(phrases) - totalPhraseReviewed
	fmt.Printf("  New: %d words, %d phrases\n", newWords, newPhrases)
	fmt.Printf("  Due now: %d\n", totalDue)

	if totalDue > 0 {
		fmt.Printf("\nRun 'lingo review start' to begin.\n")
	}
	return nil
}

func reviewStart(args []string) error {
	var limit int
	var tagFilter string
	for i := 0; i < len(args); i++ {
		if args[i] == "--limit" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &limit)
			i++
		} else if strings.HasPrefix(args[i], "--limit=") {
			fmt.Sscanf(strings.TrimPrefix(args[i], "--limit="), "%d", &limit)
		} else if args[i] == "--tag" && i+1 < len(args) {
			tagFilter = args[i+1]
			i++
		}
	}

	var tags []string
	if tagFilter != "" {
		tags = strings.Split(tagFilter, ",")
	}

	type item struct {
		kind        string
		id          string
		text        string
		def         string
		inflections string
		data        interface{} // *model.Word or *model.Phrase
	}

	// Collect due items using SQL-level filtering.
	dueWords, err := wordStore.LoadDue()
	if err != nil {
		return err
	}

	var due []item
	for i := range dueWords {
		w := &dueWords[i]
		if !store.MatchAnyTag(w.Tags, tags) {
			continue
		}
		due = append(due, item{
			kind:        "word",
			id:          w.ID,
			text:        w.Word,
			def:         formatWordDefs(w.Definitions),
			inflections: formatInflections(w.Inflections),
			data:        w,
		})
	}

	duePhrases, err := phraseStore.LoadDue()
	if err != nil {
		return err
	}

	for i := range duePhrases {
		p := &duePhrases[i]
		if !store.MatchAnyTag(p.Tags, tags) {
			continue
		}
		due = append(due, item{
			kind: "phrase",
			id:   p.ID,
			text: p.Phrase,
			def:  p.Definition,
			data: p,
		})
	}

	if len(due) == 0 {
		fmt.Println("No items due for review.")
		return nil
	}

	if limit > 0 && limit < len(due) {
		due = due[:limit]
	}

	fmt.Printf("\nStarting review session — %d items\n", len(due))
	fmt.Println("Rate: [1] Again  [2] Hard  [3] Good  [4] Easy  [q] Quit")
	fmt.Println()

	reviewed := 0
	for i := 0; i < len(due); i++ {
		it := &due[i]
		fmt.Printf("┌─ %s %d/%d ────────────────\n", it.kind, i+1, len(due))
		fmt.Printf("│ %s\n", it.text)
		fmt.Println("│")
		fmt.Print("│ Press Enter to reveal...")
		fmt.Scanln()

		fmt.Printf("│ %s\n", it.def)
		if it.inflections != "" {
			fmt.Printf("│ %s", it.inflections)
		}
		fmt.Println("│")
		fmt.Print("│ [1] Again  [2] Hard  [3] Good  [4] Easy  [q] Quit: ")

		var input string
		fmt.Scanln(&input)

		if input == "q" || input == "quit" {
			fmt.Printf("\nReviewed %d, %d remaining.\n", reviewed, len(due)-i-1)
			return nil
		}

		var rating int
		fmt.Sscanf(input, "%d", &rating)
		if rating < 1 || rating > 4 {
			fmt.Println("Invalid. Skipping.")
			continue
		}

		// Apply FSRS.
		now := time.Now().UTC()
		var days int

		switch it.kind {
		case "word":
			w := it.data.(*model.Word)
			elapsed := review.DaysLate(w.LastReviewedAt)
			if elapsed < 0 {
				elapsed = 0
			}
			var newS, newD float64
			var newState int
			newS, newD, newState, days = review.Review(
				w.Stability, w.Difficulty, w.State, elapsed, rating, review.DefaultWeights,
			)
			w.Stability = newS
			w.Difficulty = newD
			w.State = newState
			w.ReviewCount++
			w.LastReviewedAt = now
			w.NextReviewAt = review.NextReviewTime(now, days)
			w.UpdatedAt = now
			if err := wordStore.Update(*w); err != nil {
				fmt.Printf("Error saving: %v\n", err)
			}
		case "phrase":
			p := it.data.(*model.Phrase)
			elapsed := review.DaysLate(p.LastReviewedAt)
			if elapsed < 0 {
				elapsed = 0
			}
			var newS, newD float64
			var newState int
			newS, newD, newState, days = review.Review(
				p.Stability, p.Difficulty, p.State, elapsed, rating, review.DefaultWeights,
			)
			p.Stability = newS
			p.Difficulty = newD
			p.State = newState
			p.ReviewCount++
			p.LastReviewedAt = now
			p.NextReviewAt = review.NextReviewTime(now, days)
			p.UpdatedAt = now
			if err := phraseStore.Update(*p); err != nil {
				fmt.Printf("Error saving: %v\n", err)
			}
		}

		if days == 0 {
			due = append(due, *it)
			fmt.Printf("  → Again! Will come back around.\n\n")
		} else {
			reviewed++
			fmt.Printf("  → Next review in %d days\n\n", days)
		}
	}

	fmt.Printf("Done! Reviewed %d items.\n", reviewed)
	return nil
}

func formatWordDefs(defs []model.Definition) string {
	if len(defs) == 0 {
		return ""
	}
	var b strings.Builder
	n := len(defs)
	if n > 5 {
		n = 5
	}
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		pos := defs[i].Pos
		if pos == "" {
			pos = "-"
		}
		fmt.Fprintf(&b, "[%s] %s", pos, defs[i].Meaning)
	}
	return b.String()
}

func formatInflections(infs []model.Inflection) string {
	if len(infs) == 0 {
		return ""
	}
	var b strings.Builder
	for _, inf := range infs {
		b.WriteString(inf.Form)
		b.WriteString(": ")
		b.WriteString(inf.Value)
		b.WriteByte('\n')
	}
	return b.String()
}
