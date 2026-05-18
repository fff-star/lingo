package web

import (
	"fmt"
	"net/http"
	"strings"
)

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handleExportDownload(w, r)
		return
	}

	words, _ := s.Words.Load()
	phrases, _ := s.Phrases.Load()
	sentences, _ := s.Sentences.Load()
	articles, _ := s.Articles.Load()

	allTags := make(map[string]bool)
	for _, w := range words {
		for _, t := range w.Tags {
			allTags[t] = true
		}
	}
	for _, p := range phrases {
		for _, t := range p.Tags {
			allTags[t] = true
		}
	}

	var tags []string
	for t := range allTags {
		tags = append(tags, t)
	}

	s.render(w, r, "export.html", map[string]interface{}{
		"Title":         "Export",
		"WordCount":     len(words),
		"PhraseCount":   len(phrases),
		"SentenceCount": len(sentences),
		"ArticleCount":  len(articles),
		"Tags":          tags,
	})
}

func (s *Server) handleExportDownload(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	format := r.FormValue("format")
	types := r.Form["type"]
	tag := r.FormValue("tag")

	var filterTags []string
	if tag != "" {
		filterTags = strings.Split(tag, ",")
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=export.txt")

	for _, typ := range types {
		switch typ {
		case "word":
			words, _ := s.Words.Search(nil, filterTags)
			for _, wrd := range words {
				def := ""
				if len(wrd.Definitions) > 0 {
					def = wrd.Definitions[0].Meaning
				}
				if format == "csv" {
					fmt.Fprintf(w, "word,%q,%q\n", wrd.Word, def)
				} else {
					fmt.Fprintf(w, "%s\t%s\t%s\n", wrd.Word, strings.Join(wrd.Tags, ","), def)
				}
			}
		case "phrase":
			phrases, _ := s.Phrases.Search(nil, filterTags)
			for _, p := range phrases {
				if format == "csv" {
					fmt.Fprintf(w, "phrase,%q,%q\n", p.Phrase, p.Definition)
				} else {
					fmt.Fprintf(w, "%s\t%s\t%s\n", p.Phrase, strings.Join(p.Tags, ","), p.Definition)
				}
			}
		case "sentence":
			sentences, _ := s.Sentences.Search(nil, filterTags)
			for _, st := range sentences {
				if format == "csv" {
					fmt.Fprintf(w, "sentence,%q,%q\n", st.Text, st.Translation)
				} else {
					fmt.Fprintf(w, "%s\t%s\t%s\n", st.Text, strings.Join(st.Tags, ","), st.Translation)
				}
			}
		}
	}
}
