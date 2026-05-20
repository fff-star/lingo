package web

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lingo/model"
	"lingo/review"
	"lingo/store"
)

func (s *Server) handleReview(w http.ResponseWriter, r *http.Request) {
	type dueItem struct {
		ID      string
		Text    string
		Type    string
		Def     string
		AudioURL string
	}

	var due []dueItem
	dueWords, _ := s.Words.LoadDue()
	for _, w := range dueWords {
		def := formatWordDefs(w.Definitions, w.ECDictDefs)
		due = append(due, dueItem{ID: w.ID, Text: w.Word, Type: "word", Def: def, AudioURL: w.AudioURL})
	}

	duePhrases, _ := s.Phrases.LoadDue()
	for _, p := range duePhrases {
		due = append(due, dueItem{ID: p.ID, Text: p.Phrase, Type: "phrase", Def: p.Definition})
	}

	wordCount, _ := s.Words.Count()
	phraseCount, _ := s.Phrases.Count()
	todayCount, _ := s.ReviewLog.TodayCount()

	s.render(w, r, "review.html", map[string]interface{}{
		"Title":       "Review",
		"Due":         due,
		"DueCount":    len(due),
		"WordCount":   wordCount,
		"PhraseCount": phraseCount,
		"TodayCount":  todayCount,
		"DailyGoal":   20,
	})
}

// nextDue holds the identity of the next due card.
type nextDue struct {
	kind        string
	id          string
	text        string
	phonetic    string
	def         string
	inflections string
	audioURL    string
}

// findNextDue returns the first due word or phrase.
func (s *Server) findNextDue() (*nextDue, bool) {
	words, _ := s.Words.LoadDue()
	if len(words) > 0 {
		w := words[0]
		return &nextDue{
			kind:        "word",
			id:          w.ID,
			text:        w.Word,
			phonetic:    w.Phonetic,
			def:         formatWordDefs(w.Definitions, w.ECDictDefs),
			inflections: formatInflections(w.Inflections),
			audioURL:    w.AudioURL,
		}, true
	}
	phrases, _ := s.Phrases.LoadDue()
	if len(phrases) > 0 {
		p := phrases[0]
		return &nextDue{"phrase", p.ID, p.Phrase, "", p.Definition, "", ""}, true
	}
	return nil, false
}

func (s *Server) handleReviewStart(w http.ResponseWriter, r *http.Request) {
	s.renderReviewCard(w, "")
}

func (s *Server) renderReviewCard(w http.ResponseWriter, feedback string) {
	item, ok := s.findNextDue()
	if !ok {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div id="review-card" style="margin-top:2rem">`)
		fmt.Fprint(w, `<div class="review-done"><h2>All caught up!</h2><p>No more items due for review.</p><a href="/review">Back to review</a></div>`)
		fmt.Fprint(w, `</div>`)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	// The card replaces #review-card via htmx outerHTML swap.
	fmt.Fprint(w, `<div id="review-card" class="review-card" style="margin-top:2rem">`)
	if feedback != "" {
		fmt.Fprintf(w, `<p style="text-align:center;color:var(--pico-muted-color)">%s</p>`, feedback)
	}
	fmt.Fprintf(w, `<div class="front" id="front-%s">%s</div>`, item.id, html.EscapeString(item.text))
	if item.phonetic != "" {
		fmt.Fprintf(w, `<p style="text-align:center;margin-top:0.3em;opacity:0.7;font-size:0.95em">%s</p>`, html.EscapeString(item.phonetic))
	}
	if item.audioURL != "" {
		fmt.Fprintf(w, `<p style="text-align:center;margin-top:0.3em"><a href="#" onclick="new Audio('%s').play();return false" style="text-decoration:none" title="Play pronunciation">🔊</a></p>`, html.EscapeString(item.audioURL))
	}
	fmt.Fprintf(w, `<button id="show-%s" class="secondary" style="width:100%%;margin-top:1rem" onclick="document.getElementById('back-%s').hidden=false;this.hidden=true">Show Answer</button>`, item.id, item.id)
	fmt.Fprintf(w, `<div class="back" id="back-%s" hidden>`, item.id)
	if item.def != "" {
		fmt.Fprintf(w, `<p style="white-space:pre-line">%s</p>`, html.EscapeString(item.def))
	}
	if item.inflections != "" {
		fmt.Fprintf(w, `<p style="white-space:pre-line;opacity:0.7">%s</p>`, html.EscapeString(item.inflections))
	}
	fmt.Fprint(w, `<div class="rating-buttons">`)
	for i := 1; i <= 4; i++ {
		labels := []string{"", "Again", "Hard", "Good", "Easy"}
		styles := []string{"", "contrast", "secondary", "", ""}
		fmt.Fprintf(w,
			`<button hx-post="/review/rate" hx-vals='{"kind":"%s","id":"%s","rating":%d}' hx-target="#review-card" hx-swap="outerHTML" class="%s">%s</button>`,
			item.kind, item.id, i, styles[i], labels[i])
	}
	fmt.Fprint(w, `</div></div>`)
	fmt.Fprint(w, `</div>`)
}

func (s *Server) handleReviewRate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}

	r.ParseForm()
	kind := r.FormValue("kind")
	id := r.FormValue("id")
	ratingStr := r.FormValue("rating")

	rating, _ := strconv.Atoi(ratingStr)
	if rating < 1 || rating > 4 {
		http.Error(w, "invalid rating", 400)
		return
	}

	now := time.Now().UTC()
	ratingNames := []string{"", "Again", "Hard", "Good", "Easy"}
	ratingName := ratingNames[rating]

	var nextDays int
	var itemText string

	switch kind {
	case "word":
		wrd, err := s.Words.Get(id)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		elapsed := review.DaysLate(wrd.LastReviewedAt)
		if elapsed < 0 {
			elapsed = 0
		}
		newS, newD, newState, days := review.Review(
			wrd.Stability, wrd.Difficulty, wrd.State, elapsed, rating, review.DefaultWeights,
		)
		wrd.Stability = newS
		wrd.Difficulty = newD
		wrd.State = newState
		wrd.ReviewCount++
		wrd.LastReviewedAt = now
		wrd.NextReviewAt = review.NextReviewTime(now, days)
		wrd.UpdatedAt = now
		if err := s.Words.Update(*wrd); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		nextDays = days
		itemText = wrd.Word

	case "phrase":
		phr, err := s.Phrases.Get(id)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		elapsed := review.DaysLate(phr.LastReviewedAt)
		if elapsed < 0 {
			elapsed = 0
		}
		newS, newD, newState, days := review.Review(
			phr.Stability, phr.Difficulty, phr.State, elapsed, rating, review.DefaultWeights,
		)
		phr.Stability = newS
		phr.Difficulty = newD
		phr.State = newState
		phr.ReviewCount++
		phr.LastReviewedAt = now
		phr.NextReviewAt = review.NextReviewTime(now, days)
		phr.UpdatedAt = now
		if err := s.Phrases.Update(*phr); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		nextDays = days
		itemText = phr.Phrase
	}

	// Record review event for stats.
	_ = s.ReviewLog.Record(now.Format("2006-01-02"))

	feedback := fmt.Sprintf("%s — %s → %d day(s)", itemText, ratingName, nextDays)
	s.renderReviewCard(w, feedback)
}

func (s *Server) handleReviewStats(w http.ResponseWriter, r *http.Request) {
	// Collect created_at dates for "new" counts.
	var dates []time.Time
	words, _ := s.Words.Load()
	for _, w := range words {
		dates = append(dates, w.CreatedAt)
	}
	phrases, _ := s.Phrases.Load()
	for _, p := range phrases {
		dates = append(dates, p.CreatedAt)
	}

	newCounts := store.NewCountsByDate(dates)
	stats := s.ReviewLog.Stats(newCounts)

	// Find max for scaling.
	max := 1
	for i := range stats.Labels {
		if v := stats.New[i] + stats.Reviews[i]; v > max {
			max = v
		}
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<article id="review-chart"><h3>Last 15 Days</h3>`)
	fmt.Fprint(w, `<div style="display:flex;align-items:flex-end;gap:4px;height:160px;padding:0 0 1.5em 0;border-bottom:1px solid var(--pico-muted-border-color)">`)
	for i, label := range stats.Labels {
		nH := maxBar(2, stats.New[i]*140/max)
		rH := maxBar(2, stats.Reviews[i]*140/max)
		fmt.Fprintf(w, `<div style="flex:1;display:flex;align-items:flex-end;justify-content:center;gap:2px;min-width:0;position:relative">`)
		fmt.Fprintf(w, `<div style="width:45%%;height:%dpx;background:var(--pico-primary);border-radius:2px 2px 0 0" title="New: %d"></div>`, nH, stats.New[i])
		fmt.Fprintf(w, `<div style="width:45%%;height:%dpx;background:var(--pico-contrast);border-radius:2px 2px 0 0" title="Reviews: %d"></div>`, rH, stats.Reviews[i])
		fmt.Fprintf(w, `<span style="position:absolute;bottom:-1.3em;font-size:0.6em;white-space:nowrap">%s</span>`, label)
		fmt.Fprint(w, `</div>`)
	}
	fmt.Fprint(w, `</div>`)
	fmt.Fprint(w, `<div style="display:flex;gap:1em;justify-content:center;margin-top:0.5em;font-size:0.8em"><span style="color:var(--pico-primary)">New</span><span style="color:var(--pico-contrast)">Review</span></div>`)
	fmt.Fprint(w, `</article>`)
}

func maxBar(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func formatWordDefs(defs, ecDefs []model.Definition) string {
	if len(defs) == 0 && len(ecDefs) == 0 {
		return ""
	}
	var b strings.Builder
	first := true
	// ECDICT Chinese definitions (all).
	for _, d := range ecDefs {
		if !first {
			b.WriteByte('\n')
		}
		if d.Pos != "" {
			fmt.Fprintf(&b, "[%s] %s", d.Pos, d.Meaning)
		} else {
			b.WriteString(d.Meaning)
		}
		first = false
	}
	// MW English definitions (first 5).
	n := len(defs)
	if n > 5 {
		n = 5
	}
	for i := 0; i < n; i++ {
		if !first {
			b.WriteByte('\n')
		}
		pos := defs[i].Pos
		if pos == "" {
			pos = "-"
		}
		fmt.Fprintf(&b, "[%s] %s", pos, defs[i].Meaning)
		first = false
	}
	return b.String()
}

func formatInflections(inflections []model.Inflection) string {
	if len(inflections) == 0 {
		return ""
	}
	var b strings.Builder
	for _, inf := range inflections {
		b.WriteString(inf.Form)
		b.WriteString(": ")
		b.WriteString(inf.Value)
		b.WriteByte('\n')
	}
	return b.String()
}
