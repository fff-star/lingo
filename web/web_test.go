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
		"web/templates/compositions.html": {Data: []byte(`{{define "content"}}{{if .Detail}}title:{{.Detail.Title}};topic:{{.Detail.Topic}};{{if .Detail.AIAnalysis}}ai:{{.Detail.AIAnalysis.Summary}};essay:{{.Detail.AIAnalysis.ModelEssay}};{{if .Detail.AIAnalysis.ModelEssay2}}essay2:{{.Detail.AIAnalysis.ModelEssay2.Essay}};{{range .Detail.AIAnalysis.ModelEssay2.Words}}w:{{.Word}}({{.Notes}});{{end}}{{range .Detail.AIAnalysis.ModelEssay2.Phrases}}p:{{.Phrase}}({{.Notes}});{{end}}{{range .Detail.AIAnalysis.ModelEssay2.Sentences}}s:{{.Text}}({{.Why}});{{end}}{{end}}{{end}}{{else}}compositions{{end}}{{end}}`)},
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
	t.Skip("requires MW_API_KEY env var")
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

// addCompData inserts a composition with full AIAnalysis (including ModelEssay2).
func addCompData(t *testing.T, srv *Server) *model.Composition {
	t.Helper()
	now := time.Now().UTC()
	c := model.Composition{
		ID:        store.NewID("cp"),
		Title:     "My First Essay",
		Topic:     "technology and education",
		Author:    "student",
		Content:   "Technology has changed how we learn.",
		Tags:      []string{"education", "technology"},
		Notes:     "rough draft",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := srv.Compositions.Add(c); err != nil {
		t.Fatal(err)
	}
	return &c
}

// addCompDataWithAI inserts a composition with a full AIAnalysis including ModelEssay2.
func addCompDataWithAI(t *testing.T, srv *Server) *model.Composition {
	t.Helper()
	now := time.Now().UTC()
	c := model.Composition{
		ID:      store.NewID("cp"),
		Title:   "AI-Analyzed Essay",
		Topic:   "climate change",
		Author:  "student",
		Content: "Global warming is a serious issue facing humanity.",
		Tags:    []string{"environment"},
		Notes:   "",
		AIAnalysis: &model.AIAnalysis{
			Summary:       "Overall good, needs better structure.",
			SuggestedTags: []string{"climate", "argument"},
			Words: []model.ExtractedWord{
				{Word: "humanity", Definitions: []model.Definition{{Pos: "n", Meaning: "人类"}}, Example: "...", Synonyms: []string{"humankind"}, Notes: "学术词汇"},
			},
			Phrases: []model.ExtractedPhrase{
				{Phrase: "face an issue", Type: "collocation", Definition: "面对问题", Example: "...", Notes: ""},
			},
			Sentences: []model.ExtractedSentence{
				{Text: "Global warming is a serious issue.", Translation: "全球变暖是一个严重问题", Why: "简洁的论点句", SuggestedTags: []string{"writing"}},
			},
			GrammarErrors: []model.GrammarError{
				{Sentence: "bad grammar here", Correction: "good grammar here", Explanation: "语法错误", ErrorType: "grammar"},
			},
			ModelEssay: "Global warming represents one of the most pressing challenges of our era...",
			ModelEssay2: &model.ModelEssay2{
				Essay: "Climate change is a complex phenomenon driven by human activities...",
				Words: []model.ExtractedWord{
					{Word: "phenomenon", Definitions: []model.Definition{{Pos: "n", Meaning: "现象"}}, Example: "Climate change is a complex phenomenon.", Synonyms: []string{"occurrence"}, Notes: "GRE词汇，描述复杂自然或社会现象"},
				},
				Phrases: []model.ExtractedPhrase{
					{Phrase: "driven by", Type: "collocation", Definition: "由...驱动", Example: "driven by human activities", Notes: "表示因果关系的搭配"},
				},
				Sentences: []model.ExtractedSentence{
					{Text: "Rising temperatures threaten ecosystems worldwide.", Translation: "上升的气温威胁着全球生态系统", Why: "简洁有力的主题句，适合作为段落开头", SuggestedTags: []string{"topic_sentence"}},
				},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := srv.Compositions.Add(c); err != nil {
		t.Fatal(err)
	}
	return &c
}

func TestHandleCompositionDetail(t *testing.T) {
	srv := newTestServer(t)
	c := addCompData(t, srv)

	req := httptest.NewRequest("GET", "/compositions/"+c.ID, nil)
	req.SetPathValue("id", c.ID)
	rec := httptest.NewRecorder()
	srv.handleCompositionDetail(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Fatalf("status = %d, want 200", rec.Result().StatusCode)
	}
	body := rec.Body.String()

	checks := map[string]string{
		"title": "title:" + c.Title,
		"topic": "topic:" + c.Topic,
	}
	for name, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("%s: body should contain %q, got: %s", name, want, body)
		}
	}
	if strings.Contains(body, "ai:") {
		t.Errorf("body should NOT contain AI data when AIAnalysis is nil: %s", body)
	}
}

func TestHandleCompositionDetailWithAI(t *testing.T) {
	srv := newTestServer(t)
	c := addCompDataWithAI(t, srv)

	req := httptest.NewRequest("GET", "/compositions/"+c.ID, nil)
	req.SetPathValue("id", c.ID)
	rec := httptest.NewRecorder()
	srv.handleCompositionDetail(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Fatalf("status = %d, want 200", rec.Result().StatusCode)
	}
	body := rec.Body.String()

	// AI summary and model essay
	if !strings.Contains(body, "ai:"+c.AIAnalysis.Summary) {
		t.Errorf("body should contain AI summary: %s", body)
	}
	if !strings.Contains(body, "essay:"+c.AIAnalysis.ModelEssay) {
		t.Errorf("body should contain model essay: %s", body)
	}

	// ModelEssay2
	if !strings.Contains(body, "essay2:"+c.AIAnalysis.ModelEssay2.Essay) {
		t.Errorf("body should contain model essay 2: %s", body)
	}

	// Essay2 words with notes
	if !strings.Contains(body, "w:phenomenon(GRE词汇，描述复杂自然或社会现象)") {
		t.Errorf("body should contain essay2 word with notes: %s", body)
	}

	// Essay2 phrases with notes
	if !strings.Contains(body, "p:driven by(表示因果关系的搭配)") {
		t.Errorf("body should contain essay2 phrase with notes: %s", body)
	}

	// Essay2 sentences with why
	if !strings.Contains(body, "s:Rising temperatures threaten ecosystems worldwide.(简洁有力的主题句，适合作为段落开头)") {
		t.Errorf("body should contain essay2 sentence with why: %s", body)
	}
}

func TestHandleCompositionAdd(t *testing.T) {
	srv := newTestServer(t)

	form := strings.NewReader("title=Test+Essay&topic=science&author=me&content=Hello+world&tags=science,essay&notes=test+note")
	req := httptest.NewRequest("POST", "/compositions/add", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	srv.handleCompositionAdd(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Fatalf("status = %d, want 200", rec.Result().StatusCode)
	}

	// Load and verify the composition was stored with all fields.
	comps, err := srv.Compositions.Load()
	if err != nil || len(comps) == 0 {
		t.Fatal("composition should have been added")
	}
	c := comps[0]
	if c.Title != "Test Essay" {
		t.Errorf("Title = %q, want 'Test Essay'", c.Title)
	}
	if c.Topic != "science" {
		t.Errorf("Topic = %q, want 'science'", c.Topic)
	}
	if c.Author != "me" {
		t.Errorf("Author = %q, want 'me'", c.Author)
	}
	if c.Content != "Hello world" {
		t.Errorf("Content = %q, want 'Hello world'", c.Content)
	}
	if len(c.Tags) != 2 || c.Tags[0] != "science" || c.Tags[1] != "essay" {
		t.Errorf("Tags = %v, want [science essay]", c.Tags)
	}
	if c.Notes != "test note" {
		t.Errorf("Notes = %q, want 'test note'", c.Notes)
	}
}

func TestHandleCompositionUpdate(t *testing.T) {
	srv := newTestServer(t)
	c := addCompData(t, srv)

	form := strings.NewReader("title=Updated+Title&topic=updated+topic&author=new+author&content=new+content&tags=newtag&notes=new+notes")
	req := httptest.NewRequest("PUT", "/compositions/"+c.ID, form)
	req.SetPathValue("id", c.ID)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	srv.handleCompositionUpdate(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Fatalf("status = %d, want 200", rec.Result().StatusCode)
	}

	updated, err := srv.Compositions.Get(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("Title = %q", updated.Title)
	}
	if updated.Topic != "updated topic" {
		t.Errorf("Topic = %q", updated.Topic)
	}
	if updated.Author != "new author" {
		t.Errorf("Author = %q", updated.Author)
	}
	if updated.Content != "new content" {
		t.Errorf("Content = %q", updated.Content)
	}
}

func TestHandleCompositionDetailWithoutTopic(t *testing.T) {
	// Composition without topic should render without panicking.
	srv := newTestServer(t)
	now := time.Now().UTC()
	c := model.Composition{
		ID:        store.NewID("cp"),
		Title:     "No Topic Essay",
		Author:    "me",
		Content:   "Content.",
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := srv.Compositions.Add(c); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/compositions/"+c.ID, nil)
	req.SetPathValue("id", c.ID)
	rec := httptest.NewRecorder()
	srv.handleCompositionDetail(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("status = %d, want 200", rec.Result().StatusCode)
	}
	body := rec.Body.String()
	// Topic should be empty but template should still render
	if !strings.Contains(body, "title:No Topic Essay") {
		t.Errorf("body should contain title: %s", body)
	}
	// Empty topic should not cause issues
	if !strings.Contains(body, "topic:") {
		t.Errorf("topic field should appear even if empty: %s", body)
	}
}
