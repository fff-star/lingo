package web

import (
	"net/http"
	"strings"
	"time"

	"lingo/llm"
	"lingo/model"
	"lingo/store"
)

func (s *Server) handleCompositions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	tag := r.URL.Query().Get("tag")

	var keywords []string
	if q != "" {
		keywords = strings.Fields(q)
	}
	var tags []string
	if tag != "" {
		tags = strings.Split(tag, ",")
	}

	comps, err := s.Compositions.Search(keywords, tags)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	allTags, _ := s.Compositions.GetAllTags()

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "compositions.html", map[string]interface{}{
		"Title":        "Compositions",
		"Compositions": comps,
		"Query":        q,
		"Tag":          tag,
		"AllTags":      allTags,
		"Htmx":         isHtmx,
	})
}

func (s *Server) handleCompositionDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Redirect(w, r, "/compositions", 302)
		return
	}

	comp, err := s.Compositions.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	s.render(w, r, "compositions.html", map[string]interface{}{
		"Title":  comp.Title,
		"Detail": comp,
	})
}

func (s *Server) handleCompositionAdd(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "title is required", 400)
		return
	}

	now := time.Now().UTC()
	c := model.Composition{
		ID:        store.NewID("cp"),
		Title:     title,
		Author:    strings.TrimSpace(r.FormValue("author")),
		Content:   strings.TrimSpace(r.FormValue("content")),
		Tags:      []string{},
		Notes:     strings.TrimSpace(r.FormValue("notes")),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Parse tags.
	if tagsRaw := strings.TrimSpace(r.FormValue("tags")); tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				c.Tags = append(c.Tags, t)
			}
		}
	}

	if err := s.Compositions.Add(c); err != nil {
		http.Error(w, err.Error(), 409)
		return
	}

	comps, _ := s.Compositions.Search(nil, nil)
	allTags, _ := s.Compositions.GetAllTags()

	w.Header().Set("HX-Trigger", "compositionAdded")
	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "compositions.html", map[string]interface{}{
		"Title":        "Compositions",
		"Compositions": comps,
		"AllTags":      allTags,
		"Htmx":         isHtmx,
	})
}

func (s *Server) handleCompositionDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing id", 400)
		return
	}

	if err := s.Compositions.Delete(id); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	w.Header().Set("HX-Redirect", "/compositions")
	w.WriteHeader(200)
}

func (s *Server) handleCompositionUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}

	c, err := s.Compositions.Get(id)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

	c.Title = strings.TrimSpace(r.FormValue("title"))
	c.Author = strings.TrimSpace(r.FormValue("author"))
	c.Content = strings.TrimSpace(r.FormValue("content"))
	c.Notes = strings.TrimSpace(r.FormValue("notes"))
	c.UpdatedAt = time.Now().UTC()

	// Tags.
	c.Tags = nil
	if tagsRaw := strings.TrimSpace(r.FormValue("tags")); tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				c.Tags = append(c.Tags, t)
			}
		}
	}

	if err := s.Compositions.Update(*c); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "compositions.html", map[string]interface{}{
		"Title":  c.Title,
		"Detail": c,
		"Htmx":   isHtmx,
	})
}

func (s *Server) handleCompositionProcess(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cfg, err := llm.ConfigFromEnv()
	if err != nil {
		http.Error(w, "LLM not configured: "+err.Error(), 400)
		return
	}

	comp, err := s.Compositions.Get(id)
	if err != nil {
		http.Error(w, "composition not found", 404)
		return
	}

	if comp.Content == "" {
		http.Error(w, "composition has no content", 400)
		return
	}

	items, err := llm.AnalyzeComposition(cfg, comp.Content, comp.Title)
	if err != nil {
		http.Error(w, "AI analysis failed: "+err.Error(), 500)
		return
	}

	// Store analysis results inline (NOT added to word/phrase/sentence stores).
	comp.AIAnalysis = items.ToAIAnalysis()

	// Merge suggested tags.
	if len(items.SuggestedTags) > 0 {
		existing := make(map[string]bool)
		for _, t := range comp.Tags {
			existing[t] = true
		}
		for _, t := range items.SuggestedTags {
			if !existing[t] {
				comp.Tags = append(comp.Tags, t)
				existing[t] = true
			}
		}
	}

	comp.UpdatedAt = time.Now().UTC()
	if err := s.Compositions.Update(*comp); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "compositions.html", map[string]interface{}{
		"Title":  comp.Title,
		"Detail": comp,
		"Htmx":   isHtmx,
	})
}
