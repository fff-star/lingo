package web

import (
	"fmt"
	"html"
	"net/http"
	"sort"

	"lingo/review"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	wordCount, _ := s.Words.Count()
	phraseCount, _ := s.Phrases.Count()
	sentenceCount, _ := s.Sentences.Count()
	articleCount, _ := s.Articles.Count()
	compCount, _ := s.Compositions.Count()

	tags, _ := s.Tags.Load()

	// Count due reviews.
	var dueCount int
	words, _ := s.Words.Load()
	for _, w := range words {
		if review.Due(w.NextReviewAt) {
			dueCount++
		}
	}
	phrases, _ := s.Phrases.Load()
	for _, p := range phrases {
		if review.Due(p.NextReviewAt) {
			dueCount++
		}
	}

	currentStreak, longestStreak, _ := s.ReviewLog.Streak()
	todayCount, _ := s.ReviewLog.TodayCount()

	dailyGoal := 20

	todayPct := 0
	if dailyGoal > 0 {
		todayPct = todayCount * 100 / dailyGoal
		if todayPct > 100 {
			todayPct = 100
		}
	}

	s.render(w, r, "index.html", map[string]interface{}{
		"Title":          "Dashboard",
		"WordCount":      wordCount,
		"PhraseCount":    phraseCount,
		"SentenceCount":  sentenceCount,
		"ArticleCount":      articleCount,
		"CompositionCount":  compCount,
		"TagCount":          len(tags),
		"DueCount":       dueCount,
		"Tags":           tags,
		"CurrentStreak":  currentStreak,
		"LongestStreak":  longestStreak,
		"DailyGoal":      dailyGoal,
		"TodayCount":     todayCount,
		"TodayPct":       todayPct,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", 302)
}

type tagProgress struct {
	Name      string
	Total     int
	Reviewed  int
	Due       int
	Mastered  int
	AvgStab   float64
}

func (s *Server) handleStatsTags(w http.ResponseWriter, r *http.Request) {
	tagMap := make(map[string]*tagProgress)

	// Seed with all known tags so empty-tag rows still appear.
	tags, _ := s.Tags.Load()
	for _, t := range tags {
		tagMap[t.Name] = &tagProgress{Name: t.Name}
	}

	words, _ := s.Words.Load()
	for _, w := range words {
		for _, t := range w.Tags {
			if _, ok := tagMap[t]; !ok {
				tagMap[t] = &tagProgress{Name: t}
			}
			p := tagMap[t]
			p.Total++
			if w.ReviewCount > 0 {
				p.Reviewed++
				p.AvgStab += w.Stability
			}
			if review.Due(w.NextReviewAt) {
				p.Due++
			}
			if w.State == 2 && w.Difficulty <= 4.0 {
				p.Mastered++
			}
		}
	}

	phrases, _ := s.Phrases.Load()
	for _, ph := range phrases {
		for _, t := range ph.Tags {
			if _, ok := tagMap[t]; !ok {
				tagMap[t] = &tagProgress{Name: t}
			}
			p := tagMap[t]
			p.Total++
			if ph.ReviewCount > 0 {
				p.Reviewed++
				p.AvgStab += ph.Stability
			}
			if review.Due(ph.NextReviewAt) {
				p.Due++
			}
			if ph.State == 2 && ph.Difficulty <= 4.0 {
				p.Mastered++
			}
		}
	}

	// Compute avg stability.
	for _, p := range tagMap {
		if p.Reviewed > 0 {
			p.AvgStab = p.AvgStab / float64(p.Reviewed)
		}
	}

	// Sort by total descending.
	var progress []tagProgress
	for _, p := range tagMap {
		progress = append(progress, *p)
	}
	sort.Slice(progress, func(i, j int) bool {
		return progress[i].Total > progress[j].Total
	})

	w.Header().Set("Content-Type", "text/html")
	if len(progress) == 0 {
		fmt.Fprint(w, `<p>No tags yet.</p>`)
		return
	}
	fmt.Fprint(w, `<h3>Progress by Tag</h3><table class="striped"><thead><tr><th>Tag</th><th>Total</th><th>Reviewed</th><th>Due</th><th>Mastered</th><th>Avg Stability</th></tr></thead><tbody>`)
	for _, p := range progress {
		masteredPct := 0
		if p.Total > 0 {
			masteredPct = p.Mastered * 100 / p.Total
		}
		fmt.Fprintf(w, `<tr><td><span class="tag">%s</span></td><td>%d</td><td>%d</td><td>%d</td><td>%d (%d%%)</td><td>%.1f days</td></tr>`,
			html.EscapeString(p.Name), p.Total, p.Reviewed, p.Due, p.Mastered, masteredPct, p.AvgStab)
	}
	fmt.Fprint(w, `</tbody></table>`)
}
