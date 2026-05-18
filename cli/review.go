package cli

import (
	"fmt"
	"strings"
	"time"

	"lingo/model"
	"lingo/review"
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

	var dueWords int
	var totalReviewed int
	for _, w := range words {
		if w.ReviewCount > 0 {
			totalReviewed++
		}
		if review.Due(w.NextReviewAt) {
			dueWords++
		}
	}

	phrases, err := phraseStore.Load()
	if err != nil {
		return err
	}

	var duePhrases int
	var totalPhraseReviewed int
	for _, p := range phrases {
		if p.ReviewCount > 0 {
			totalPhraseReviewed++
		}
		if review.Due(p.NextReviewAt) {
			duePhrases++
		}
	}

	fmt.Println("═══════ Review Stats ═══════")
	fmt.Println()
	fmt.Printf("  Words:   %d due, %d reviewed, %d total\n", dueWords, totalReviewed, len(words))
	fmt.Printf("  Phrases: %d due, %d reviewed, %d total\n", duePhrases, totalPhraseReviewed, len(phrases))
	fmt.Println()
	totalDue := dueWords + duePhrases
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

	// Collect due words.
	words, err := wordStore.Load()
	if err != nil {
		return err
	}

	type item struct {
		kind string
		id   string
		text string
		def  string
		data interface{} // *model.Word or *model.Phrase
	}

	var due []item
	for i := range words {
		w := &words[i]
		if !review.Due(w.NextReviewAt) {
			continue
		}
		if !matchAnyTagStr(w.Tags, tags) {
			continue
		}
		due = append(due, item{
			kind: "word",
			id:   w.ID,
			text: w.Word,
			def:  formatWordDefs(w.Definitions),
			data: w,
		})
	}

	phrases, err := phraseStore.Load()
	if err != nil {
		return err
	}

	for i := range phrases {
		p := &phrases[i]
		if !review.Due(p.NextReviewAt) {
			continue
		}
		if !matchAnyTagStr(p.Tags, tags) {
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
	for _, it := range due {
		fmt.Printf("┌─ %s %d/%d ────────────────\n", it.kind, reviewed+1, len(due))
		fmt.Printf("│ %s\n", it.text)
		fmt.Println("│")
		fmt.Print("│ Press Enter to reveal...")
		fmt.Scanln()

		fmt.Printf("│ %s\n", it.def)
		fmt.Println("│")
		fmt.Print("│ [1] Again  [2] Hard  [3] Good  [4] Easy  [q] Quit: ")

		var input string
		fmt.Scanln(&input)

		if input == "q" || input == "quit" {
			fmt.Printf("\nReviewed %d, %d remaining.\n", reviewed, len(due)-reviewed)
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
			w.NextReviewAt = review.NextDayStart(now, days)
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
			p.NextReviewAt = review.NextDayStart(now, days)
			p.UpdatedAt = now
			if err := phraseStore.Update(*p); err != nil {
				fmt.Printf("Error saving: %v\n", err)
			}
		}

		reviewed++
		fmt.Printf("  → Next review in %d days\n\n", days)
	}

	fmt.Printf("Done! Reviewed %d items.\n", reviewed)
	return nil
}

func matchAnyTagStr(tags []string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, ft := range filter {
		for _, t := range tags {
			if t == ft {
				return true
			}
		}
	}
	return false
}

func formatWordDefs(defs []model.Definition) string {
	if len(defs) == 0 {
		return ""
	}
	var b strings.Builder
	for i, d := range defs {
		if i > 0 {
			b.WriteByte('\n')
		}
		pos := d.Pos
		if pos == "" {
			pos = "-"
		}
		fmt.Fprintf(&b, "[%s] %s", pos, d.Meaning)
	}
	return b.String()
}
