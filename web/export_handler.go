package web

import (
	"fmt"
	"net/http"
	"strings"

	"lingo/export"
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
	comps, _ := s.Compositions.Search(nil, nil)

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
		"Title":           "Export",
		"WordCount":       len(words),
		"PhraseCount":     len(phrases),
		"SentenceCount":   len(sentences),
		"ArticleCount":    len(articles),
		"CompositionCount": len(comps),
		"Tags":            tags,
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

	// For tex/pdf formats, we only support a single type at a time.
	if format == "tex" || format == "pdf" {
		if len(types) == 0 {
			http.Error(w, "select at least one type", 400)
			return
		}
		s.handleLatexExport(w, types[0], format, filterTags, tag)
		return
	}

	// Legacy text/csv export.
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

func (s *Server) handleLatexExport(w http.ResponseWriter, typ, format string, filterTags []string, tagFilter string) {
	var texContent string
	var filename string
	var err error

	switch typ {
	case "word":
		words, _ := s.Words.Search(nil, filterTags)
		texContent, err = export.GenerateWordsTeX(words, tagFilter)
		filename = "lingo-words"
	case "article":
		articles, _ := s.Articles.Search(nil, filterTags)
		if len(articles) == 0 {
			http.Error(w, "no articles found", 404)
			return
		}
		if len(articles) == 1 {
			texContent, err = export.GenerateArticleTeX(articles[0])
		} else {
			texContent, err = export.GenerateArticlesTeX(articles, tagFilter)
		}
		filename = "lingo-articles"
	case "composition":
		comps, _ := s.Compositions.Search(nil, filterTags)
		if len(comps) == 0 {
			http.Error(w, "no compositions found", 404)
			return
		}
		if len(comps) == 1 {
			texContent, err = export.GenerateCompositionTeX(comps[0])
		} else {
			texContent, err = export.GenerateCompositionsTeX(comps, tagFilter)
		}
		filename = "lingo-compositions"
	default:
		http.Error(w, "unsupported type for LaTeX export: "+typ, 400)
		return
	}

	if err != nil {
		http.Error(w, "generate tex: "+err.Error(), 500)
		return
	}

	if format == "pdf" {
		pdf, err := export.CompilePDF(texContent)
		if err != nil {
			// Fall back to .tex on compilation failure.
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename="+filename+".tex")
			w.Write([]byte(texContent))
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".pdf")
		w.Write(pdf)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename+".tex")
	w.Write([]byte(texContent))
}
