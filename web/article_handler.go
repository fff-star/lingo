package web

import (
	"net/http"
	"strings"
	"time"

	"lingo/llm"
	"lingo/model"
	"lingo/store"
)

func (s *Server) handleArticles(w http.ResponseWriter, r *http.Request) {
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

	articles, err := s.Articles.Search(keywords, tags)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	allTags, _ := s.Articles.GetAllTags()

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "articles.html", map[string]interface{}{
		"Title":    "Articles",
		"Articles": articles,
		"Query":    q,
		"Tag":      tag,
		"AllTags":  allTags,
		"Htmx":     isHtmx,
	})
}

func (s *Server) handleArticleDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Redirect(w, r, "/articles", 302)
		return
	}

	article, err := s.Articles.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	s.render(w, r, "articles.html", map[string]interface{}{
		"Title":  article.Title,
		"Detail": article,
	})
}

func (s *Server) handleArticleAdd(w http.ResponseWriter, r *http.Request) {
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
	a := model.Article{
		ID:        store.NewID("ar"),
		Title:     title,
		Author:    strings.TrimSpace(r.FormValue("author")),
		Source:    strings.TrimSpace(r.FormValue("source")),
		SourceURL: strings.TrimSpace(r.FormValue("source_url")),
		Content:   strings.TrimSpace(r.FormValue("content")),
		Summary:   strings.TrimSpace(r.FormValue("summary")),
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
				a.Tags = append(a.Tags, t)
			}
		}
	}

	if err := s.Articles.Add(a); err != nil {
		http.Error(w, err.Error(), 409)
		return
	}

	articles, _ := s.Articles.Search(nil, nil)
	allTags, _ := s.Articles.GetAllTags()

	w.Header().Set("HX-Trigger", "articleAdded")
	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "articles.html", map[string]interface{}{
		"Title":    "Articles",
		"Articles": articles,
		"AllTags":  allTags,
		"Htmx":     isHtmx,
	})
}

func (s *Server) handleArticleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing id", 400)
		return
	}

	if err := s.Articles.Delete(id); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	w.Header().Set("HX-Redirect", "/articles")
	w.WriteHeader(200)
}

func (s *Server) handleArticleUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}

	a, err := s.Articles.Get(id)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

	a.Title = strings.TrimSpace(r.FormValue("title"))
	a.Author = strings.TrimSpace(r.FormValue("author"))
	a.Source = strings.TrimSpace(r.FormValue("source"))
	a.SourceURL = strings.TrimSpace(r.FormValue("source_url"))
	a.Content = strings.TrimSpace(r.FormValue("content"))
	a.Notes = strings.TrimSpace(r.FormValue("notes"))
	a.UpdatedAt = time.Now().UTC()

	// Tags.
	a.Tags = nil
	if tagsRaw := strings.TrimSpace(r.FormValue("tags")); tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				a.Tags = append(a.Tags, t)
			}
		}
	}

	if err := s.Articles.Update(*a); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "articles.html", map[string]interface{}{
		"Title":  a.Title,
		"Detail": a,
		"Htmx":   isHtmx,
	})
}

func (s *Server) handleArticleProcess(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cfg, err := llm.ConfigFromEnv()
	if err != nil {
		http.Error(w, "LLM not configured: "+err.Error(), 400)
		return
	}

	a, err := s.Articles.Get(id)
	if err != nil {
		http.Error(w, "article not found", 404)
		return
	}

	if a.Content == "" {
		http.Error(w, "article has no content", 400)
		return
	}

	items, err := llm.ProcessArticle(cfg, a.Content, a.Title)
	if err != nil {
		http.Error(w, "LLM processing failed: "+err.Error(), 500)
		return
	}

	// Store analysis inline — no additions to word/phrase/sentence stores.
	a.AIAnalysis = items.ToAIAnalysis()

	// Merge suggested tags.
	if len(items.SuggestedTags) > 0 {
		existing := make(map[string]bool)
		for _, t := range a.Tags {
			existing[t] = true
		}
		for _, t := range items.SuggestedTags {
			if !existing[t] {
				a.Tags = append(a.Tags, t)
				existing[t] = true
			}
		}
	}
	a.UpdatedAt = time.Now().UTC()
	_ = s.Articles.Update(*a)

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "articles.html", map[string]interface{}{
		"Title":  a.Title,
		"Detail": a,
		"Htmx":   isHtmx,
	})
}
