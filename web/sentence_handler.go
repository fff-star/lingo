package web

import (
	"net/http"
	"strings"
	"time"

	"lingo/model"
	"lingo/store"
)

func (s *Server) handleSentences(w http.ResponseWriter, r *http.Request) {
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

	sentences, err := s.Sentences.Search(keywords, tags)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	allTags, _ := s.Sentences.GetAllTags()

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "sentences.html", map[string]interface{}{
		"Title":     "Sentences",
		"Sentences": sentences,
		"Query":     q,
		"Tag":       tag,
		"AllTags":   allTags,
		"Htmx":      isHtmx,
	})
}

func (s *Server) handleSentenceDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Redirect(w, r, "/sentences", 302)
		return
	}

	sentence, err := s.Sentences.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	s.render(w, r, "sentences.html", map[string]interface{}{
		"Title":  "Sentence",
		"Detail": sentence,
	})
}

func (s *Server) handleSentenceBatch(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}
	action := r.FormValue("action")
	tagName := strings.TrimSpace(r.FormValue("tag"))
	idsRaw := r.FormValue("ids")
	var ids []string
	for _, id := range strings.Split(idsRaw, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 || tagName == "" {
		http.Error(w, "ids and tag are required", 400)
		return
	}
	sentences, _ := s.Sentences.Load()
	for i := range sentences {
		st := &sentences[i]
		if !store.HasString(ids, st.ID) {
			continue
		}
		switch action {
		case "tag":
			if store.HasString(st.Tags, tagName) {
				continue
			}
			st.Tags = append(st.Tags, tagName)
		case "untag":
			if !store.HasString(st.Tags, tagName) {
				continue
			}
			st.Tags = store.RemoveString(st.Tags, tagName)
		default:
			http.Error(w, "unknown action: "+action, 400)
			return
		}
		st.UpdatedAt = time.Now().UTC()
		if err := s.Sentences.Update(*st); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
	allSentences, _ := s.Sentences.Search(nil, nil)
	allTags, _ := s.Sentences.GetAllTags()
	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "sentences.html", map[string]interface{}{
		"Title":     "Sentences",
		"Sentences": allSentences,
		"AllTags":   allTags,
		"Htmx":      isHtmx,
	})
}

func (s *Server) handleSentenceAdd(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}

	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		http.Error(w, "text is required", 400)
		return
	}

	now := time.Now().UTC()
	st := model.Sentence{
		ID:          store.NewID("st"),
		Text:        text,
		Source:      strings.TrimSpace(r.FormValue("source")),
		SourceURL:   strings.TrimSpace(r.FormValue("source_url")),
		Author:      strings.TrimSpace(r.FormValue("author")),
		Translation: strings.TrimSpace(r.FormValue("translation")),
		Tags:        []string{},
		Notes:       strings.TrimSpace(r.FormValue("notes")),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Parse tags.
	if tagsRaw := strings.TrimSpace(r.FormValue("tags")); tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				st.Tags = append(st.Tags, t)
			}
		}
	}

	if err := s.Sentences.Add(st); err != nil {
		http.Error(w, err.Error(), 409)
		return
	}

	sentences, _ := s.Sentences.Search(nil, nil)
	allTags, _ := s.Sentences.GetAllTags()

	w.Header().Set("HX-Trigger", "sentenceAdded")
	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "sentences.html", map[string]interface{}{
		"Title":     "Sentences",
		"Sentences": sentences,
		"AllTags":   allTags,
		"Htmx":      isHtmx,
	})
}

func (s *Server) handleSentenceDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing id", 400)
		return
	}

	if err := s.Sentences.Delete(id); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	w.Header().Set("HX-Redirect", "/sentences")
	w.WriteHeader(200)
}

func (s *Server) handleSentenceUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}

	st, err := s.Sentences.Get(id)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

	st.Text = strings.TrimSpace(r.FormValue("text"))
	st.Source = strings.TrimSpace(r.FormValue("source"))
	st.SourceURL = strings.TrimSpace(r.FormValue("source_url"))
	st.Author = strings.TrimSpace(r.FormValue("author"))
	st.Translation = strings.TrimSpace(r.FormValue("translation"))
	st.Notes = strings.TrimSpace(r.FormValue("notes"))
	st.UpdatedAt = time.Now().UTC()

	// Tags.
	st.Tags = nil
	if tagsRaw := strings.TrimSpace(r.FormValue("tags")); tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				st.Tags = append(st.Tags, t)
			}
		}
	}

	if err := s.Sentences.Update(*st); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "sentences.html", map[string]interface{}{
		"Title":  "Sentence",
		"Detail": st,
		"Htmx":   isHtmx,
	})
}
