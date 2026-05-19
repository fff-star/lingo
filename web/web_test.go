package web

import (
	"database/sql"
	"io/fs"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"lingo/model"
	"lingo/store"
)

// fakeTemplateFS returns an embed.FS-like fs.FS with the minimum templates needed.
func fakeTemplateFS() fs.FS {
	m := fstest.MapFS{
		"web/templates/base.html":         {Data: []byte(`{{define "base.html"}}<html>{{block "content" .}}{{end}}</html>{{end}}`)},
		"web/templates/index.html":        {Data: []byte(`{{define "content"}}index:{{.WordCount}}:{{.DueCount}}{{end}}`)},
		"web/templates/words.html":        {Data: []byte(`{{define "content"}}words:{{.Words}}{{end}}`)},
		"web/templates/phrases.html":      {Data: []byte(`{{define "content"}}phrases{{end}}`)},
		"web/templates/sentences.html":    {Data: []byte(`{{define "content"}}sentences{{end}}`)},
		"web/templates/articles.html":     {Data: []byte(`{{define "content"}}articles{{end}}`)},
		"web/templates/compositions.html": {Data: []byte(`{{define "content"}}compositions{{end}}`)},
		"web/templates/review.html":       {Data: []byte(`{{define "content"}}review:{{.DueCount}}{{end}}`)},
		"web/templates/export.html":       {Data: []byte(`{{define "content"}}export{{end}}`)},
	}
	return m
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	if err := store.CreateTables(db); err != nil {
		t.Fatal(err)
	}

	srv, err := New(
		store.NewWordStore(db),
		store.NewPhraseStore(db),
		store.NewSentenceStore(db),
		store.NewArticleStore(db),
		store.NewCompositionStore(db),
		store.NewTagStore(db),
		store.NewReviewLog(db),
		fakeTemplateFS(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func addTestData(t *testing.T, srv *Server) {
	t.Helper()
	now := time.Now().UTC()
	srv.Words.Add(model.Word{
		ID: store.NewID("wd"), Word: "ephemeral",
		Phonetic:    "/ɪˈfem.ər.əl/",
		Definitions: []model.Definition{{Pos: "adj", Meaning: "short-lived"}},
		Synonyms:    []string{}, Advanced: []string{}, Examples: []string{},
		Inflections: []model.Inflection{}, Tags: []string{"gre"},
		NextReviewAt: now, CreatedAt: now, UpdatedAt: now,
	})
	srv.Phrases.Add(model.Phrase{
		ID: store.NewID("ph"), Phrase: "in the long run",
		Definition: "eventually", Tags: []string{"writing"},
		Synonyms: []string{}, Advanced: []string{}, Examples: []string{},
		NextReviewAt: now, CreatedAt: now, UpdatedAt: now,
	})
}

// ---- Handler tests ----

func TestHandleIndex(t *testing.T) {
	srv := newTestServer(t)
	addTestData(t, srv)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	srv.handleIndex(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "index:1:") {
		t.Errorf("index should show word count 1: %s", body)
	}
}

func TestHandleWords(t *testing.T) {
	srv := newTestServer(t)
	addTestData(t, srv)

	req := httptest.NewRequest("GET", "/words", nil)
	rec := httptest.NewRecorder()
	srv.handleWords(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestHandleWordsTagFilter(t *testing.T) {
	srv := newTestServer(t)
	addTestData(t, srv)

	req := httptest.NewRequest("GET", "/words?tag=gre", nil)
	rec := httptest.NewRecorder()
	srv.handleWords(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestHandleWordDetail(t *testing.T) {
	srv := newTestServer(t)
	addTestData(t, srv)

	words, _ := srv.Words.Load()
	req := httptest.NewRequest("GET", "/words/"+words[0].ID, nil)
	req.SetPathValue("id", words[0].ID)
	rec := httptest.NewRecorder()
	srv.handleWordDetail(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestHandleWordDetailNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/words/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleWordDetail(rec, req)

	if rec.Result().StatusCode != 404 {
		t.Errorf("status = %d, want 404", rec.Result().StatusCode)
	}
}

func TestHandleWordCheckFound(t *testing.T) {
	srv := newTestServer(t)
	addTestData(t, srv)

	req := httptest.NewRequest("GET", "/words/check?q=ephemeral", nil)
	rec := httptest.NewRecorder()
	srv.handleWordCheck(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"found":true`) {
		t.Errorf("should find ephemeral: %s", body)
	}
}

func TestHandleWordCheckNotFound(t *testing.T) {
	t.Skip("requires network access to Free Dictionary API")
}

func TestHandleWordDelete(t *testing.T) {
	srv := newTestServer(t)
	addTestData(t, srv)

	words, _ := srv.Words.Load()
	req := httptest.NewRequest("DELETE", "/words/"+words[0].ID, nil)
		req.SetPathValue("id", words[0].ID)
	rec := httptest.NewRecorder()
	srv.handleWordDelete(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
	// Verify redirect header.
	if rec.Header().Get("HX-Redirect") != "/words" {
		t.Errorf("HX-Redirect = %q, want /words", rec.Header().Get("HX-Redirect"))
	}
}

// ---- Phrase handler tests ----

func TestHandlePhrases(t *testing.T) {
	srv := newTestServer(t)
	addTestData(t, srv)

	req := httptest.NewRequest("GET", "/phrases", nil)
	rec := httptest.NewRecorder()
	srv.handlePhrases(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestHandlePhraseDetailNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/phrases/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handlePhraseDetail(rec, req)

	if rec.Result().StatusCode != 404 {
		t.Errorf("status = %d, want 404", rec.Result().StatusCode)
	}
}

// ---- Review handler tests ----

func TestHandleReview(t *testing.T) {
	srv := newTestServer(t)
	addTestData(t, srv)

	req := httptest.NewRequest("GET", "/review", nil)
	rec := httptest.NewRecorder()
	srv.handleReview(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "review:") {
		t.Errorf("review page should render: %s", body)
	}
}

func TestHandleReviewStartAllCaughtUp(t *testing.T) {
	srv := newTestServer(t)
	// No items due — should show "All caught up!"

	req := httptest.NewRequest("GET", "/review/start", nil)
	rec := httptest.NewRecorder()
	srv.handleReviewStart(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
	if !strings.Contains(rec.Body.String(), "All caught up") {
		t.Errorf("should show 'All caught up': %s", rec.Body.String())
	}
}

func TestHandleReviewRateInvalidRating(t *testing.T) {
	srv := newTestServer(t)

	form := strings.NewReader("kind=word&id=nonexistent&rating=5")
	req := httptest.NewRequest("POST", "/review/rate", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	srv.handleReviewRate(rec, req)

	if rec.Result().StatusCode != 400 {
		t.Errorf("status = %d, want 400 for invalid rating", rec.Result().StatusCode)
	}
}

// ---- Index handler ----

func TestHandleIndexNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/nonexistent-page", nil)
	rec := httptest.NewRecorder()
	srv.handleIndex(rec, req)

	if rec.Result().StatusCode != 404 {
		t.Errorf("status = %d, want 404", rec.Result().StatusCode)
	}
}

// ---- Stats handler redirect ----

func TestHandleStatsRedirect(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/stats", nil)
	rec := httptest.NewRecorder()
	srv.handleStats(rec, req)

	if rec.Result().StatusCode != 302 {
		t.Errorf("status = %d, want 302", rec.Result().StatusCode)
	}
}

// ---- SSE setup ----

func TestSetupSSE(t *testing.T) {
	rec := httptest.NewRecorder()
	send, err := setupSSE(rec)
	if err != nil {
		t.Fatalf("setupSSE: %v", err)
	}
	if send == nil {
		t.Fatal("send function should not be nil")
	}
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", rec.Header().Get("Content-Type"))
	}

	// Call send — should write SSE-formatted data.
	send("test", "hello")
	body := rec.Body.String()
	if !strings.Contains(body, "event: test") {
		t.Errorf("SSE body should contain event: %s", body)
	}
	if !strings.Contains(body, "data: hello") {
		t.Errorf("SSE body should contain data: %s", body)
	}
}

// ---- Sentence handler tests ----

func TestHandleSentences(t *testing.T) {
	srv := newTestServer(t)
	addTestData(t, srv)

	req := httptest.NewRequest("GET", "/sentences", nil)
	rec := httptest.NewRecorder()
	srv.handleSentences(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestHandleSentenceDetailNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/sentences/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleSentenceDetail(rec, req)

	if rec.Result().StatusCode != 404 {
		t.Errorf("status = %d, want 404", rec.Result().StatusCode)
	}
}

// ---- Article handler tests ----

func TestHandleArticles(t *testing.T) {
	srv := newTestServer(t)
	addTestData(t, srv)

	req := httptest.NewRequest("GET", "/articles", nil)
	rec := httptest.NewRecorder()
	srv.handleArticles(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestHandleArticleDetailNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/articles/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleArticleDetail(rec, req)

	if rec.Result().StatusCode != 404 {
		t.Errorf("status = %d, want 404", rec.Result().StatusCode)
	}
}

// ---- Composition handler tests ----

func TestHandleCompositions(t *testing.T) {
	srv := newTestServer(t)
	addTestData(t, srv)

	req := httptest.NewRequest("GET", "/compositions", nil)
	rec := httptest.NewRecorder()
	srv.handleCompositions(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestHandleCompositionDetailNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/compositions/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.handleCompositionDetail(rec, req)

	if rec.Result().StatusCode != 404 {
		t.Errorf("status = %d, want 404", rec.Result().StatusCode)
	}
}
