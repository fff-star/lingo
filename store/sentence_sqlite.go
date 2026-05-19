package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"lingo/model"
)

type sqliteSentenceStore struct {
	db *sql.DB
}

func NewSentenceStore(db *sql.DB) SentenceStore {
	return &sqliteSentenceStore{db: db}
}

func (s *sqliteSentenceStore) Load() ([]model.Sentence, error) {
	rows, err := s.db.Query(`SELECT id, text, source, source_url, author, translation,
		tags, notes, created_at, updated_at FROM sentences ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSentences(rows)
}

func (s *sqliteSentenceStore) Add(st model.Sentence) error {
	_, err := s.db.Exec(`INSERT INTO sentences
		(id, text, source, source_url, author, translation, tags, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		st.ID, st.Text, st.Source, st.SourceURL, st.Author, st.Translation,
		jsonArr(st.Tags), st.Notes,
		fmtTime(st.CreatedAt), fmtTime(st.UpdatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrExists
		}
		return err
	}
	return nil
}

func (s *sqliteSentenceStore) Get(idPrefix string) (*model.Sentence, error) {
	st, err := scanSentence(s.db.QueryRow(`SELECT id, text, source, source_url, author, translation,
		tags, notes, created_at, updated_at FROM sentences WHERE id = ?`,
		idPrefix))
	if err == nil {
		return &st, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	rows, err := s.db.Query(`SELECT id, text, source, source_url, author, translation,
		tags, notes, created_at, updated_at FROM sentences WHERE id LIKE ? LIMIT 1`,
		idPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sentences, err := scanSentences(rows)
	if err != nil {
		return nil, err
	}
	if len(sentences) == 0 {
		return nil, ErrNotFound
	}
	return &sentences[0], nil
}

func (s *sqliteSentenceStore) Update(st model.Sentence) error {
	result, err := s.db.Exec(`UPDATE sentences SET text=?, source=?, source_url=?, author=?, translation=?,
		tags=?, notes=?, updated_at=? WHERE id=?`,
		st.Text, st.Source, st.SourceURL, st.Author, st.Translation,
		jsonArr(st.Tags), st.Notes, fmtTime(st.UpdatedAt), st.ID,
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

func (s *sqliteSentenceStore) Delete(idPrefix string) error {
	result, err := s.db.Exec(`DELETE FROM sentences WHERE id LIKE ?`,
		idPrefix+"%")
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteSentenceStore) Search(keywords []string, tags []string) ([]model.Sentence, error) {
	if len(keywords) > 0 {
		if results, ok := s.ftsSearch(keywords, tags); ok {
			return results, nil
		}
	}

	query := `SELECT id, text, source, source_url, author, translation,
		tags, notes, created_at, updated_at FROM sentences WHERE 1=1`
	var args []interface{}

	for _, kw := range keywords {
		kw = "%" + strings.ToLower(kw) + "%"
		query += ` AND (LOWER(text) LIKE ? OR LOWER(translation) LIKE ? OR LOWER(source) LIKE ? OR LOWER(notes) LIKE ?)`
		args = append(args, kw, kw, kw, kw)
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
	return scanSentences(rows)
}

func (s *sqliteSentenceStore) ftsSearch(keywords, tags []string) ([]model.Sentence, bool) {
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

	sql := `SELECT s.id, s.text, s.source, s.source_url, s.author, s.translation,
		s.tags, s.notes, s.created_at, s.updated_at
		FROM sentences s
		INNER JOIN sentences_fts f ON s.rowid = f.rowid
		WHERE sentences_fts MATCH ?`
	var args []interface{}
	args = append(args, ftsQuery)

	for _, tag := range tags {
		sql += ` AND s.tags LIKE ?`
		args = append(args, fmt.Sprintf(`%%"%s"%%`, tag))
	}
	sql += ` ORDER BY rank`

	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	results, err := scanSentences(rows)
	if err != nil || len(results) == 0 {
		return nil, false
	}
	return results, true
}

func (s *sqliteSentenceStore) GetAllTags() ([]string, error) {
	sentences, err := s.Load()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var tags []string
	for _, st := range sentences {
		for _, t := range st.Tags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	sort.Strings(tags)
	return tags, nil
}

func (s *sqliteSentenceStore) Count() (int, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sentences").Scan(&n)
	return n, err
}

func scanSentence(row interface{ Scan(...interface{}) error }) (model.Sentence, error) {
	var st model.Sentence
	var tgs, ca, ua string
	err := row.Scan(
		&st.ID, &st.Text, &st.Source, &st.SourceURL, &st.Author, &st.Translation,
		&tgs, &st.Notes, &ca, &ua,
	)
	if err != nil {
		return st, err
	}
	json.Unmarshal([]byte(tgs), &st.Tags)
	st.CreatedAt = parseTime(ca)
	st.UpdatedAt = parseTime(ua)
	return st, nil
}

func scanSentences(rows *sql.Rows) ([]model.Sentence, error) {
	var sentences []model.Sentence
	for rows.Next() {
		st, err := scanSentence(rows)
		if err != nil {
			return nil, err
		}
		sentences = append(sentences, st)
	}
	return sentences, rows.Err()
}
