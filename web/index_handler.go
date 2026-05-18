package web

import (
	"net/http"

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

	s.render(w, r, "index.html", map[string]interface{}{
		"Title":         "Dashboard",
		"WordCount":     wordCount,
		"PhraseCount":   phraseCount,
		"SentenceCount": sentenceCount,
		"ArticleCount":  articleCount,
		"TagCount":      len(tags),
		"DueCount":      dueCount,
		"Tags":          tags,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", 302)
}
