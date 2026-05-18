package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"lingo/dict"
	"lingo/model"
	"lingo/review"
	"lingo/store"
)

func (s *Server) handleWords(w http.ResponseWriter, r *http.Request) {
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

	words, err := s.Words.Search(keywords, tags)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	allTags, _ := s.Words.GetAllTags()

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "words.html", map[string]interface{}{
		"Title":   "Words",
		"Words":   words,
		"Query":   q,
		"Tag":     tag,
		"AllTags": allTags,
		"Htmx":    isHtmx,
	})
}

func (s *Server) handleWordDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Redirect(w, r, "/words", 302)
		return
	}

	word, err := s.Words.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	s.render(w, r, "words.html", map[string]interface{}{
		"Title":  word.Word,
		"Detail": word,
	})
}

func (s *Server) handleWordAdd(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}

	word := strings.TrimSpace(r.FormValue("word"))
	if word == "" {
		http.Error(w, "word is required", 400)
		return
	}

	now := time.Now().UTC()
	stability, difficulty, state := review.NewCardState(review.DefaultWeights)
	wm := model.Word{
		ID:          store.NewID("wd"),
		Word:        word,
		Phonetic:    strings.TrimSpace(r.FormValue("phonetic")),
		Definitions: []model.Definition{},
		Examples:    []string{},
		Inflections: []model.Inflection{},
		Synonyms:    []string{},
		Advanced:    []string{},
		Tags:        []string{},
		Notes:       strings.TrimSpace(r.FormValue("notes")),
		Stability:   stability,
		Difficulty:  difficulty,
		State:       state,
		NextReviewAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Parse definitions from form: pos|meaning per line.
	if defsRaw := strings.TrimSpace(r.FormValue("definitions")); defsRaw != "" {
		for _, line := range strings.Split(defsRaw, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			pos, meaning := "", line
			if idx := strings.Index(line, "|"); idx >= 0 {
				pos = strings.TrimSpace(line[:idx])
				meaning = strings.TrimSpace(line[idx+1:])
			}
			if meaning != "" {
				wm.Definitions = append(wm.Definitions, model.Definition{Pos: pos, Meaning: meaning})
			}
		}
	}

	// Parse examples.
	if exRaw := strings.TrimSpace(r.FormValue("examples")); exRaw != "" {
		for _, line := range strings.Split(exRaw, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				wm.Examples = append(wm.Examples, line)
			}
		}
	}

	// Parse tags.
	if tagsRaw := strings.TrimSpace(r.FormValue("tags")); tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				wm.Tags = append(wm.Tags, t)
			}
		}
	}

	// LLM inflection is handled by the lookup endpoint; the save path
	// stays fast by skipping LLM calls here.

	if err := s.Words.Add(wm); err != nil {
		http.Error(w, err.Error(), 409)
		return
	}

	// Render the updated list.
	words, _ := s.Words.Search(nil, nil)
	allTags, _ := s.Words.GetAllTags()

	w.Header().Set("HX-Trigger", "wordAdded")
	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "words.html", map[string]interface{}{
		"Title":   "Words",
		"Words":   words,
		"AllTags": allTags,
		"Htmx":    isHtmx,
	})
}

func (s *Server) handleWordDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing id", 400)
		return
	}

	if err := s.Words.Delete(id); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	w.Header().Set("HX-Redirect", "/words")
	w.WriteHeader(200)
}

func (s *Server) handleWordUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}

	wm, err := s.Words.Get(id)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

	wm.Word = strings.TrimSpace(r.FormValue("word"))
	wm.Phonetic = strings.TrimSpace(r.FormValue("phonetic"))
	wm.Notes = strings.TrimSpace(r.FormValue("notes"))
	wm.UpdatedAt = time.Now().UTC()

	// Definitions.
	wm.Definitions = nil
	if defsRaw := strings.TrimSpace(r.FormValue("definitions")); defsRaw != "" {
		for _, line := range strings.Split(defsRaw, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			pos, meaning := "", line
			if idx := strings.Index(line, "|"); idx >= 0 {
				pos = strings.TrimSpace(line[:idx])
				meaning = strings.TrimSpace(line[idx+1:])
			}
			if meaning != "" {
				wm.Definitions = append(wm.Definitions, model.Definition{Pos: pos, Meaning: meaning})
			}
		}
	}

	// Examples.
	wm.Examples = nil
	if exRaw := strings.TrimSpace(r.FormValue("examples")); exRaw != "" {
		for _, line := range strings.Split(exRaw, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				wm.Examples = append(wm.Examples, line)
			}
		}
	}

	// Synonyms.
	wm.Synonyms = nil
	if sRaw := strings.TrimSpace(r.FormValue("synonyms")); sRaw != "" {
		for _, s := range strings.Split(sRaw, ",") {
			if s = strings.TrimSpace(s); s != "" {
				wm.Synonyms = append(wm.Synonyms, s)
			}
		}
	}

	// Advanced.
	wm.Advanced = nil
	if aRaw := strings.TrimSpace(r.FormValue("advanced")); aRaw != "" {
		for _, a := range strings.Split(aRaw, ",") {
			if a = strings.TrimSpace(a); a != "" {
				wm.Advanced = append(wm.Advanced, a)
			}
		}
	}

	// Tags.
	wm.Tags = nil
	if tagsRaw := strings.TrimSpace(r.FormValue("tags")); tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				wm.Tags = append(wm.Tags, t)
			}
		}
	}

	if err := s.Words.Update(*wm); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	isHtmx := r.Header.Get("HX-Request") == "true"
	s.render(w, r, "words.html", map[string]interface{}{
		"Title":  wm.Word,
		"Detail": wm,
		"Htmx":   isHtmx,
	})
}

func (s *Server) handleWordLookup(w http.ResponseWriter, r *http.Request) {
	word := r.URL.Query().Get("q")
	if word == "" {
		word = r.URL.Query().Get("word")
	}
	if word == "" {
		http.Error(w, "missing q param", 400)
		return
	}

	info, err := dict.Lookup(word)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
