package web

import (
	"net/http"
	"strings"
	"time"

	"lingo/model"
	"lingo/review"
	"lingo/store"
)

func (s *Server) handlePhrases(w http.ResponseWriter, r *http.Request) {
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

	phrases, err := s.Phrases.Search(keywords, tags)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	allTags, _ := s.Phrases.GetAllTags()

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "phrases.html", map[string]interface{}{
		"Title":   "Phrases",
		"Phrases": phrases,
		"Query":   q,
		"Tag":     tag,
		"AllTags": allTags,
		"Htmx":    isHtmx,
	})
}

func (s *Server) handlePhraseDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Redirect(w, r, "/phrases", 302)
		return
	}

	phrase, err := s.Phrases.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	s.render(w, r, "phrases.html", map[string]interface{}{
		"Title":  phrase.Phrase,
		"Detail": phrase,
	})
}

func (s *Server) handlePhraseBatch(w http.ResponseWriter, r *http.Request) {
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
	phrases, _ := s.Phrases.Load()
	for i := range phrases {
		p := &phrases[i]
		if !store.HasString(ids, p.ID) {
			continue
		}
		switch action {
		case "tag":
			if store.HasString(p.Tags, tagName) {
				continue
			}
			p.Tags = append(p.Tags, tagName)
		case "untag":
			if !store.HasString(p.Tags, tagName) {
				continue
			}
			p.Tags = store.RemoveString(p.Tags, tagName)
		default:
			http.Error(w, "unknown action: "+action, 400)
			return
		}
		p.UpdatedAt = time.Now().UTC()
		if err := s.Phrases.Update(*p); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
	allPhrases, _ := s.Phrases.Search(nil, nil)
	allTags, _ := s.Phrases.GetAllTags()
	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "phrases.html", map[string]interface{}{
		"Title":   "Phrases",
		"Phrases": allPhrases,
		"AllTags": allTags,
		"Htmx":    isHtmx,
	})
}

func (s *Server) handlePhraseAdd(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}

	phrase := strings.TrimSpace(r.FormValue("phrase"))
	if phrase == "" {
		http.Error(w, "phrase is required", 400)
		return
	}

	now := time.Now().UTC()
	stability, difficulty, state := review.NewCardState(review.DefaultWeights)
	typ := strings.TrimSpace(r.FormValue("type"))
	if typ == "" {
		typ = "other"
	}

	pm := model.Phrase{
		ID:         store.NewID("ph"),
		Phrase:     phrase,
		Type:       typ,
		Words:      []string{},
		Definition: strings.TrimSpace(r.FormValue("definition")),
		Examples:   []string{},
		Synonyms:   []string{},
		Advanced:   []string{},
		Tags:       []string{},
		Notes:      strings.TrimSpace(r.FormValue("notes")),
		Stability:   stability,
		Difficulty:  difficulty,
		State:       state,
		NextReviewAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Parse words.
	if wordsRaw := strings.TrimSpace(r.FormValue("words")); wordsRaw != "" {
		for _, w := range strings.Fields(wordsRaw) {
			pm.Words = append(pm.Words, w)
		}
	}

	// Parse examples.
	if exRaw := strings.TrimSpace(r.FormValue("examples")); exRaw != "" {
		for _, line := range strings.Split(exRaw, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				pm.Examples = append(pm.Examples, line)
			}
		}
	}

	// Parse tags.
	if tagsRaw := strings.TrimSpace(r.FormValue("tags")); tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				pm.Tags = append(pm.Tags, t)
			}
		}
	}

	if err := s.Phrases.Add(pm); err != nil {
		http.Error(w, err.Error(), 409)
		return
	}

	phrases, _ := s.Phrases.Search(nil, nil)
	allTags, _ := s.Phrases.GetAllTags()

	w.Header().Set("HX-Trigger", "phraseAdded")
	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "phrases.html", map[string]interface{}{
		"Title":   "Phrases",
		"Phrases": phrases,
		"AllTags": allTags,
		"Htmx":    isHtmx,
	})
}

func (s *Server) handlePhraseDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing id", 400)
		return
	}

	if err := s.Phrases.Delete(id); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	w.Header().Set("HX-Redirect", "/phrases")
	w.WriteHeader(200)
}

func (s *Server) handlePhraseUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}

	pm, err := s.Phrases.Get(id)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

	pm.Phrase = strings.TrimSpace(r.FormValue("phrase"))
	pm.Type = strings.TrimSpace(r.FormValue("type"))
	if pm.Type == "" {
		pm.Type = "other"
	}
	pm.Definition = strings.TrimSpace(r.FormValue("definition"))
	pm.Notes = strings.TrimSpace(r.FormValue("notes"))
	pm.UpdatedAt = time.Now().UTC()

	// Words.
	pm.Words = nil
	if wordsRaw := strings.TrimSpace(r.FormValue("words")); wordsRaw != "" {
		for _, w := range strings.Fields(wordsRaw) {
			pm.Words = append(pm.Words, w)
		}
	}

	// Examples.
	pm.Examples = nil
	if exRaw := strings.TrimSpace(r.FormValue("examples")); exRaw != "" {
		for _, line := range strings.Split(exRaw, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				pm.Examples = append(pm.Examples, line)
			}
		}
	}

	// Synonyms.
	pm.Synonyms = nil
	if sRaw := strings.TrimSpace(r.FormValue("synonyms")); sRaw != "" {
		for _, s := range strings.Split(sRaw, ",") {
			if s = strings.TrimSpace(s); s != "" {
				pm.Synonyms = append(pm.Synonyms, s)
			}
		}
	}

	// Advanced.
	pm.Advanced = nil
	if aRaw := strings.TrimSpace(r.FormValue("advanced")); aRaw != "" {
		for _, a := range strings.Split(aRaw, ",") {
			if a = strings.TrimSpace(a); a != "" {
				pm.Advanced = append(pm.Advanced, a)
			}
		}
	}

	// Tags.
	pm.Tags = nil
	if tagsRaw := strings.TrimSpace(r.FormValue("tags")); tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				pm.Tags = append(pm.Tags, t)
			}
		}
	}

	if err := s.Phrases.Update(*pm); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "phrases.html", map[string]interface{}{
		"Title":  pm.Phrase,
		"Detail": pm,
		"Htmx":   isHtmx,
	})
}
