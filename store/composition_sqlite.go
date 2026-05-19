package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"lingo/model"
)

type sqliteCompositionStore struct {
	db *sql.DB
}

func NewCompositionStore(db *sql.DB) CompositionStore {
	return &sqliteCompositionStore{db: db}
}

func (s *sqliteCompositionStore) Load() ([]model.Composition, error) {
	rows, err := s.db.Query(`SELECT id, title, author, content, tags, notes, ai_analysis,
		created_at, updated_at FROM compositions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCompositions(rows)
}

func (s *sqliteCompositionStore) Add(c model.Composition) error {
	var aiStr *string
	if c.AIAnalysis != nil {
		b, _ := json.Marshal(c.AIAnalysis)
		s := string(b)
		aiStr = &s
	}
	_, err := s.db.Exec(`INSERT INTO compositions
		(id, title, author, content, tags, notes, ai_analysis, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Title, c.Author, c.Content,
		jsonArr(c.Tags), c.Notes, aiStr,
		fmtTime(c.CreatedAt), fmtTime(c.UpdatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrExists
		}
		return err
	}
	return nil
}

func (s *sqliteCompositionStore) Get(idPrefix string) (*model.Composition, error) {
	c, err := scanComposition(s.db.QueryRow(`SELECT id, title, author, content, tags, notes, ai_analysis,
		created_at, updated_at FROM compositions WHERE id = ?`,
		idPrefix))
	if err == nil {
		return &c, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	rows, err := s.db.Query(`SELECT id, title, author, content, tags, notes, ai_analysis,
		created_at, updated_at FROM compositions WHERE id LIKE ? LIMIT 1`,
		idPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comps, err := scanCompositions(rows)
	if err != nil {
		return nil, err
	}
	if len(comps) == 0 {
		return nil, ErrNotFound
	}
	return &comps[0], nil
}

func (s *sqliteCompositionStore) Update(c model.Composition) error {
	var aiStr *string
	if c.AIAnalysis != nil {
		b, _ := json.Marshal(c.AIAnalysis)
		s := string(b)
		aiStr = &s
	}
	result, err := s.db.Exec(`UPDATE compositions SET title=?, author=?, content=?, tags=?, notes=?,
		ai_analysis=?, updated_at=? WHERE id=?`,
		c.Title, c.Author, c.Content, jsonArr(c.Tags), c.Notes, aiStr,
		fmtTime(c.UpdatedAt), c.ID,
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

func (s *sqliteCompositionStore) Delete(idPrefix string) error {
	result, err := s.db.Exec(`DELETE FROM compositions WHERE id LIKE ?`, idPrefix+"%")
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteCompositionStore) Search(keywords []string, tags []string) ([]model.Composition, error) {
	if len(keywords) > 0 {
		if results, ok := s.ftsSearch(keywords, tags); ok {
			return results, nil
		}
	}

	// For search, exclude the heavy content column.
	query := `SELECT id, title, author, '' as content, tags, notes, ai_analysis,
		created_at, updated_at FROM compositions WHERE 1=1`
	var args []interface{}

	for _, kw := range keywords {
		kw = "%" + strings.ToLower(kw) + "%"
		query += ` AND (LOWER(title) LIKE ? OR LOWER(content) LIKE ?)`
		args = append(args, kw, kw)
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
	return scanCompositions(rows)
}

func (s *sqliteCompositionStore) ftsSearch(keywords, tags []string) ([]model.Composition, bool) {
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

	sql := `SELECT c.id, c.title, c.author, '' as content, c.tags, c.notes, c.ai_analysis,
		c.created_at, c.updated_at
		FROM compositions c
		INNER JOIN compositions_fts f ON c.rowid = f.rowid
		WHERE compositions_fts MATCH ?`
	var args []interface{}
	args = append(args, ftsQuery)

	for _, tag := range tags {
		sql += ` AND c.tags LIKE ?`
		args = append(args, fmt.Sprintf(`%%"%s"%%`, tag))
	}
	sql += ` ORDER BY rank`

	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	results, err := scanCompositions(rows)
	if err != nil || len(results) == 0 {
		return nil, false
	}
	return results, true
}

func (s *sqliteCompositionStore) GetAllTags() ([]string, error) {
	comps, err := s.Load()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var tags []string
	for _, c := range comps {
		for _, t := range c.Tags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	sort.Strings(tags)
	return tags, nil
}

func (s *sqliteCompositionStore) Count() (int, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(*) FROM compositions").Scan(&n)
	return n, err
}

func scanComposition(row interface{ Scan(...interface{}) error }) (model.Composition, error) {
	var c model.Composition
	var tgs string
	var aiStr *string
	var ca, ua string
	err := row.Scan(
		&c.ID, &c.Title, &c.Author, &c.Content,
		&tgs, &c.Notes, &aiStr, &ca, &ua,
	)
	if err != nil {
		return c, err
	}
	json.Unmarshal([]byte(tgs), &c.Tags)
	if aiStr != nil && *aiStr != "" {
		var ai model.AIAnalysis
		if json.Unmarshal([]byte(*aiStr), &ai) == nil {
			c.AIAnalysis = &ai
		}
	}
	c.CreatedAt = parseTime(ca)
	c.UpdatedAt = parseTime(ua)
	return c, nil
}

func scanCompositions(rows *sql.Rows) ([]model.Composition, error) {
	var comps []model.Composition
	for rows.Next() {
		c, err := scanComposition(rows)
		if err != nil {
			return nil, err
		}
		comps = append(comps, c)
	}
	return comps, rows.Err()
}
