package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"lingo/store"
)

var templateFuncs = template.FuncMap{
	"join": func(ss []string, sep string) string { return strings.Join(ss, sep) },
}

type Server struct {
	Words        store.WordStore
	Phrases      store.PhraseStore
	Sentences    store.SentenceStore
	Articles     store.ArticleStore
	Compositions store.CompositionStore
	Tags         store.TagStore
	ReviewLog    store.ReviewLog
	templates    map[string]*template.Template
}

func New(ws store.WordStore, ps store.PhraseStore, ss store.SentenceStore, as store.ArticleStore, cs store.CompositionStore, ts store.TagStore, rl store.ReviewLog, tmplFS fs.FS) (*Server, error) {
	srv := &Server{
		Words:        ws,
		Phrases:      ps,
		Sentences:    ss,
		Articles:     as,
		Compositions: cs,
		Tags:         ts,
		ReviewLog:    rl,
	}

	// Parse base + each page template from embedded FS so the binary is self-contained.
	pages := []string{"index", "words", "phrases", "sentences", "articles", "compositions", "review", "export"}
	srv.templates = make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		tmpl, err := template.New("base.html").Funcs(templateFuncs).ParseFS(
			tmplFS,
			"web/templates/base.html",
			"web/templates/"+page+".html",
		)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", page, err)
		}
		srv.templates[page+".html"] = tmpl
	}

	return srv, nil
}

func (s *Server) Register(mux *http.ServeMux, staticFS fs.FS) {
	mux.HandleFunc("/", s.handleIndex)
	// Words
	mux.HandleFunc("GET /words", s.handleWords)
	mux.HandleFunc("GET /words/lookup", s.handleWordLookup)
	mux.HandleFunc("GET /words/check", s.handleWordCheck)
	mux.HandleFunc("GET /words/suggest", s.handleWordSuggest)
	mux.HandleFunc("GET /words/lookup-panel", s.handleWordLookupPanel)
	mux.HandleFunc("POST /words/quick-add", s.handleWordQuickAdd)
	mux.HandleFunc("POST /words/add", s.handleWordAdd)
	mux.HandleFunc("POST /words/batch", s.handleWordBatch)
	mux.HandleFunc("GET /words/{id}", s.handleWordDetail)
	mux.HandleFunc("PUT /words/{id}", s.handleWordUpdate)
	mux.HandleFunc("DELETE /words/{id}", s.handleWordDelete)
	// Phrases
	mux.HandleFunc("GET /phrases", s.handlePhrases)
	mux.HandleFunc("POST /phrases/add", s.handlePhraseAdd)
	mux.HandleFunc("POST /phrases/batch", s.handlePhraseBatch)
	mux.HandleFunc("GET /phrases/{id}", s.handlePhraseDetail)
	mux.HandleFunc("PUT /phrases/{id}", s.handlePhraseUpdate)
	mux.HandleFunc("DELETE /phrases/{id}", s.handlePhraseDelete)
	// Sentences
	mux.HandleFunc("GET /sentences", s.handleSentences)
	mux.HandleFunc("POST /sentences/add", s.handleSentenceAdd)
	mux.HandleFunc("POST /sentences/batch", s.handleSentenceBatch)
	mux.HandleFunc("GET /sentences/{id}", s.handleSentenceDetail)
	mux.HandleFunc("PUT /sentences/{id}", s.handleSentenceUpdate)
	mux.HandleFunc("DELETE /sentences/{id}", s.handleSentenceDelete)
	// Articles
	mux.HandleFunc("GET /articles", s.handleArticles)
	mux.HandleFunc("POST /articles/add", s.handleArticleAdd)
	mux.HandleFunc("POST /articles/batch", s.handleArticleBatch)
	mux.HandleFunc("GET /articles/{id}", s.handleArticleDetail)
	mux.HandleFunc("PUT /articles/{id}", s.handleArticleUpdate)
	mux.HandleFunc("DELETE /articles/{id}", s.handleArticleDelete)
	mux.HandleFunc("POST /articles/{id}/process", s.handleArticleProcess)
	mux.HandleFunc("GET /articles/{id}/process-stream", s.handleArticleProcessSSE)
	// Compositions
	mux.HandleFunc("GET /compositions", s.handleCompositions)
	mux.HandleFunc("POST /compositions/add", s.handleCompositionAdd)
	mux.HandleFunc("GET /compositions/{id}", s.handleCompositionDetail)
	mux.HandleFunc("PUT /compositions/{id}", s.handleCompositionUpdate)
	mux.HandleFunc("DELETE /compositions/{id}", s.handleCompositionDelete)
	mux.HandleFunc("POST /compositions/{id}/process", s.handleCompositionProcess)
	mux.HandleFunc("GET /compositions/{id}/process-stream", s.handleCompositionProcessSSE)
	// Review
	mux.HandleFunc("/review", s.handleReview)
	mux.HandleFunc("/review/start", s.handleReviewStart)
	mux.HandleFunc("/review/rate", s.handleReviewRate)
	mux.HandleFunc("/review/stats", s.handleReviewStats)
	// Export
	mux.HandleFunc("/export", s.handleExport)
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/stats/tags", s.handleStatsTags)

	// Static files (embedded).
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("static files not embedded: " + err.Error())
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, name string, data interface{}) {
	tmpl := s.templates[name]
	if tmpl == nil {
		http.Error(w, "template not found: "+name, 500)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
			http.Error(w, err.Error(), 500)
		}
	} else {
		if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
			http.Error(w, err.Error(), 500)
		}
	}
}

func (s *Server) MustLoad() {
	if _, err := s.Words.Load(); err != nil {
		log.Printf("load words: %v", err)
	}
	if _, err := s.Phrases.Load(); err != nil {
		log.Printf("load phrases: %v", err)
	}
	if _, err := s.Sentences.Load(); err != nil {
		log.Printf("load sentences: %v", err)
	}
	if _, err := s.Articles.Load(); err != nil {
		log.Printf("load articles: %v", err)
	}
	if _, err := s.Compositions.Load(); err != nil {
		log.Printf("load compositions: %v", err)
	}
	if _, err := s.Tags.Load(); err != nil {
		log.Printf("load tags: %v", err)
	}
}

func setupSSE(w http.ResponseWriter) (func(event, data string), error) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return nil, fmt.Errorf("streaming not supported")
	}
	return func(event, data string) {
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}, nil
}
