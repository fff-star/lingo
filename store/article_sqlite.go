package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"lingo/model"
)

type sqliteArticleStore struct {
	db *sql.DB
}

func NewArticleStore(db *sql.DB) ArticleStore {
	return &sqliteArticleStore{db: db}
}

func (s *sqliteArticleStore) Load() ([]model.Article, error) {
	rows, err := s.db.Query(`SELECT id, title, author, source, source_url, content, summary,
		tags, notes, ai_analysis, created_at, updated_at FROM articles ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArticles(rows)
}

func (s *sqliteArticleStore) Add(a model.Article) error {
	var aiStr *string
	if a.AIAnalysis != nil {
		b, _ := json.Marshal(a.AIAnalysis)
		s := string(b)
		aiStr = &s
	}
	_, err := s.db.Exec(`INSERT INTO articles
		(id, title, author, source, source_url, content, summary, tags, notes, ai_analysis, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Title, a.Author, a.Source, a.SourceURL, a.Content, a.Summary,
		jsonArr(a.Tags), a.Notes, aiStr,
		fmtTime(a.CreatedAt), fmtTime(a.UpdatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrExists
		}
		return err
	}
	return nil
}

func (s *sqliteArticleStore) Get(idPrefix string) (*model.Article, error) {
	a, err := scanArticle(s.db.QueryRow(`SELECT id, title, author, source, source_url, content, summary,
		tags, notes, ai_analysis, created_at, updated_at FROM articles WHERE id = ?`,
		idPrefix))
	if err == nil {
		return &a, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	rows, err := s.db.Query(`SELECT id, title, author, source, source_url, content, summary,
		tags, notes, ai_analysis, created_at, updated_at FROM articles WHERE id LIKE ? LIMIT 1`,
		idPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles, err := scanArticles(rows)
	if err != nil {
		return nil, err
	}
	if len(articles) == 0 {
		return nil, ErrNotFound
	}
	return &articles[0], nil
}

func (s *sqliteArticleStore) Update(a model.Article) error {
	var aiStr *string
	if a.AIAnalysis != nil {
		b, _ := json.Marshal(a.AIAnalysis)
		s := string(b)
		aiStr = &s
	}
	result, err := s.db.Exec(`UPDATE articles SET title=?, author=?, source=?, source_url=?, content=?, summary=?,
		tags=?, notes=?, ai_analysis=?, updated_at=? WHERE id=?`,
		a.Title, a.Author, a.Source, a.SourceURL, a.Content, a.Summary,
		jsonArr(a.Tags), a.Notes, aiStr, fmtTime(a.UpdatedAt), a.ID,
	)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteArticleStore) Delete(idPrefix string) error {
	result, err := s.db.Exec(`DELETE FROM articles WHERE id LIKE ?`, idPrefix+"%")
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteArticleStore) Search(keywords []string, tags []string) ([]model.Article, error) {
	if len(keywords) > 0 {
		if results, ok := s.ftsSearch(keywords, tags); ok {
			return results, nil
		}
	}

	// For search, exclude the heavy content column.
	query := `SELECT id, title, author, source, source_url, '' as content, summary,
		tags, notes, ai_analysis, created_at, updated_at FROM articles WHERE 1=1`
	var args []interface{}

	for _, kw := range keywords {
		kw = "%" + strings.ToLower(kw) + "%"
		query += ` AND (LOWER(title) LIKE ? OR LOWER(summary) LIKE ? OR LOWER(content) LIKE ?)`
		args = append(args, kw, kw, kw)
	}
	for _, tag := range tags {
		query += ` AND tags LIKE ?`
		args = append(args, fmt.Sprintf(`%%"%s"%%`, tag))
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArticles(rows)
}

func (s *sqliteArticleStore) ftsSearch(keywords, tags []string) ([]model.Article, bool) {
	var ftsTerms []string
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		ftsTerms = append(ftsTerms, `"`+strings.ReplaceAll(kw, `"`, `""`)+`"*`)
	}
	if len(ftsTerms) == 0 {
		return nil, false
	}
	ftsQuery := strings.Join(ftsTerms, " ")

	sql := `SELECT a.id, a.title, a.author, a.source, a.source_url, '' as content, a.summary,
		a.tags, a.notes, a.ai_analysis, a.created_at, a.updated_at
		FROM articles a
		INNER JOIN articles_fts f ON a.rowid = f.rowid
		WHERE articles_fts MATCH ?`
	var args []interface{}
	args = append(args, ftsQuery)

	for _, tag := range tags {
		sql += ` AND a.tags LIKE ?`
		args = append(args, fmt.Sprintf(`%%"%s"%%`, tag))
	}
	sql += ` ORDER BY rank`

	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	results, err := scanArticles(rows)
	if err != nil || len(results) == 0 {
		return nil, false
	}
	return results, true
}

func (s *sqliteArticleStore) GetAllTags() ([]string, error) {
	articles, err := s.Load()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var tags []string
	for _, a := range articles {
		for _, t := range a.Tags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	sort.Strings(tags)
	return tags, nil
}

func (s *sqliteArticleStore) Count() (int, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&n)
	return n, err
}

func scanArticle(row interface{ Scan(...interface{}) error }) (model.Article, error) {
	var a model.Article
	var tgs string
	var aiStr *string
	var ca, ua string
	err := row.Scan(
		&a.ID, &a.Title, &a.Author, &a.Source, &a.SourceURL, &a.Content, &a.Summary,
		&tgs, &a.Notes, &aiStr, &ca, &ua,
	)
	if err != nil {
		return a, err
	}
	json.Unmarshal([]byte(tgs), &a.Tags)
	if aiStr != nil && *aiStr != "" {
		var ai model.AIAnalysis
		if json.Unmarshal([]byte(*aiStr), &ai) == nil {
			a.AIAnalysis = &ai
		}
	}
	a.CreatedAt = parseTime(ca)
	a.UpdatedAt = parseTime(ua)
	return a, nil
}

func scanArticles(rows *sql.Rows) ([]model.Article, error) {
	var articles []model.Article
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, rows.Err()
}
