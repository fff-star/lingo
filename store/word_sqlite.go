package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"lingo/model"
)

type sqliteWordStore struct {
	db *sql.DB
}

func NewWordStore(db *sql.DB) WordStore {
	return &sqliteWordStore{db: db}
}

func (s *sqliteWordStore) Load() ([]model.Word, error) {
	rows, err := s.db.Query(`SELECT id, word, phonetic, audio_url, definitions, examples, inflections,
		synonyms, advanced, tags, notes, review_count,
		last_reviewed_at, next_review_at, stability, difficulty, state,
		created_at, updated_at FROM words ORDER BY word`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWords(rows)
}

func (s *sqliteWordStore) LoadDue() ([]model.Word, error) {
	now := fmtTime(time.Now().UTC())
	rows, err := s.db.Query(`SELECT id, word, phonetic, audio_url, definitions, examples, inflections,
		synonyms, advanced, tags, notes, review_count,
		last_reviewed_at, next_review_at, stability, difficulty, state,
		created_at, updated_at FROM words
		WHERE next_review_at <= ? OR next_review_at = '0001-01-01T00:00:00Z'
		ORDER BY next_review_at`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWords(rows)
}

func (s *sqliteWordStore) Save(words []model.Word) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM words"); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO words
		(id, word, phonetic, audio_url, definitions, examples, inflections, synonyms, advanced, tags, notes,
		 review_count, last_reviewed_at, next_review_at, stability, difficulty, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, w := range words {
		if err := insertWord(stmt, w); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *sqliteWordStore) Add(w model.Word) error {
	_, err := s.db.Exec(`INSERT INTO words
		(id, word, phonetic, audio_url, definitions, examples, inflections, synonyms, advanced, tags, notes,
		 review_count, last_reviewed_at, next_review_at, stability, difficulty, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.Word, w.Phonetic, w.AudioURL,
		jsonArr(w.Definitions), jsonArr(w.Examples), jsonArr(w.Inflections),
		jsonArr(w.Synonyms), jsonArr(w.Advanced), jsonArr(w.Tags),
		w.Notes, w.ReviewCount,
		fmtTime(w.LastReviewedAt), fmtTime(w.NextReviewAt),
		w.Stability, w.Difficulty, w.State,
		fmtTime(w.CreatedAt), fmtTime(w.UpdatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrExists
		}
		return err
	}
	return nil
}

func (s *sqliteWordStore) Get(idPrefix string) (*model.Word, error) {
	// Try exact ID match first.
	w, err := scanWord(s.db.QueryRow(`SELECT id, word, phonetic, audio_url, definitions, examples, inflections,
		synonyms, advanced, tags, notes, review_count,
		last_reviewed_at, next_review_at, stability, difficulty, state,
		created_at, updated_at FROM words WHERE id = ? OR word = ?`,
		idPrefix, idPrefix))
	if err == nil {
		return &w, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Try prefix match.
	rows, err := s.db.Query(`SELECT id, word, phonetic, audio_url, definitions, examples, inflections,
		synonyms, advanced, tags, notes, review_count,
		last_reviewed_at, next_review_at, stability, difficulty, state,
		created_at, updated_at FROM words WHERE id LIKE ? OR word LIKE ? LIMIT 1`,
		idPrefix+"%", idPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	words, err := scanWords(rows)
	if err != nil {
		return nil, err
	}
	if len(words) == 0 {
		return nil, ErrNotFound
	}
	return &words[0], nil
}

func (s *sqliteWordStore) Update(w model.Word) error {
	result, err := s.db.Exec(`UPDATE words SET word=?, phonetic=?, audio_url=?, definitions=?, examples=?, inflections=?,
		synonyms=?, advanced=?, tags=?, notes=?, review_count=?,
		last_reviewed_at=?, next_review_at=?, stability=?, difficulty=?, state=?,
		updated_at=? WHERE id=?`,
		w.Word, w.Phonetic, w.AudioURL,
		jsonArr(w.Definitions), jsonArr(w.Examples), jsonArr(w.Inflections),
		jsonArr(w.Synonyms), jsonArr(w.Advanced), jsonArr(w.Tags),
		w.Notes, w.ReviewCount,
		fmtTime(w.LastReviewedAt), fmtTime(w.NextReviewAt),
		w.Stability, w.Difficulty, w.State,
		fmtTime(w.UpdatedAt),
		w.ID,
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

func (s *sqliteWordStore) Delete(idPrefix string) error {
	result, err := s.db.Exec(`DELETE FROM words WHERE id LIKE ? OR word = ?`,
		idPrefix+"%", idPrefix)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteWordStore) Search(keywords []string, tags []string) ([]model.Word, error) {
	// Try FTS5 first when keywords are provided.
	if len(keywords) > 0 {
		if results, ok := s.ftsSearch(keywords, tags); ok {
			return results, nil
		}
		// FTS returned nothing — fall back to LIKE for substring matching.
	}

	query := `SELECT id, word, phonetic, audio_url, definitions, examples, inflections,
		synonyms, advanced, tags, notes, review_count,
		last_reviewed_at, next_review_at, stability, difficulty, state,
		created_at, updated_at FROM words WHERE 1=1`
	var args []interface{}

	for _, kw := range keywords {
		kw = "%" + strings.ToLower(kw) + "%"
		query += ` AND (LOWER(word) LIKE ? OR LOWER(definitions) LIKE ? OR LOWER(synonyms) LIKE ? OR LOWER(advanced) LIKE ?)`
		args = append(args, kw, kw, kw, kw)
	}
	for _, tag := range tags {
		query += ` AND tags LIKE ?`
		args = append(args, fmt.Sprintf(`%%"%s"%%`, tag))
	}
	query += ` ORDER BY word`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWords(rows)
}

func (s *sqliteWordStore) ftsSearch(keywords, tags []string) ([]model.Word, bool) {
	// Build FTS5 query: each keyword gets prefix wildcard, joined by implicit AND.
	var ftsTerms []string
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		// Quote the term for FTS5, append * for prefix matching.
		ftsTerms = append(ftsTerms, `"`+strings.ReplaceAll(kw, `"`, `""`)+`"*`)
	}
	if len(ftsTerms) == 0 {
		return nil, false
	}
	ftsQuery := strings.Join(ftsTerms, " ")

	sql := `SELECT w.id, w.word, w.phonetic, w.definitions, w.examples, w.inflections,
		w.synonyms, w.advanced, w.tags, w.notes, w.review_count,
		w.last_reviewed_at, w.next_review_at, w.stability, w.difficulty, w.state,
		w.created_at, w.updated_at
		FROM words w
		INNER JOIN words_fts f ON w.rowid = f.rowid
		WHERE words_fts MATCH ?`
	var args []interface{}
	args = append(args, ftsQuery)

	for _, tag := range tags {
		sql += ` AND w.tags LIKE ?`
		args = append(args, fmt.Sprintf(`%%"%s"%%`, tag))
	}
	sql += ` ORDER BY rank`

	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	results, err := scanWords(rows)
	if err != nil || len(results) == 0 {
		return nil, false
	}
	return results, true
}

func (s *sqliteWordStore) AllIDs() (map[string]string, error) {
	rows, err := s.db.Query("SELECT id, word FROM words ORDER BY word")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var id, word string
		if err := rows.Scan(&id, &word); err != nil {
			return nil, err
		}
		m[id] = word
	}
	return m, rows.Err()
}

func (s *sqliteWordStore) GetAllTags() ([]string, error) {
	words, err := s.Load()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var tags []string
	for _, w := range words {
		for _, t := range w.Tags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	sort.Strings(tags)
	return tags, nil
}

func (s *sqliteWordStore) Count() (int, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(*) FROM words").Scan(&n)
	return n, err
}

func scanWord(row interface{ Scan(...interface{}) error }) (model.Word, error) {
	var w model.Word
	var defs, exs, infs, syns, advs, tgs string
	var lra, nra, ca, ua string
	err := row.Scan(
		&w.ID, &w.Word, &w.Phonetic, &w.AudioURL,
		&defs, &exs, &infs, &syns, &advs, &tgs,
		&w.Notes,
		&w.ReviewCount, &lra, &nra,
		&w.Stability, &w.Difficulty, &w.State,
		&ca, &ua,
	)
	if err != nil {
		return w, err
	}
	json.Unmarshal([]byte(defs), &w.Definitions)
	json.Unmarshal([]byte(exs), &w.Examples)
	json.Unmarshal([]byte(infs), &w.Inflections)
	json.Unmarshal([]byte(syns), &w.Synonyms)
	json.Unmarshal([]byte(advs), &w.Advanced)
	json.Unmarshal([]byte(tgs), &w.Tags)
	w.LastReviewedAt = parseTime(lra)
	w.NextReviewAt = parseTime(nra)
	w.CreatedAt = parseTime(ca)
	w.UpdatedAt = parseTime(ua)
	return w, nil
}

func scanWords(rows *sql.Rows) ([]model.Word, error) {
	var words []model.Word
	for rows.Next() {
		w, err := scanWord(rows)
		if err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	return words, rows.Err()
}

func insertWord(stmt *sql.Stmt, w model.Word) error {
	_, err := stmt.Exec(
		w.ID, w.Word, w.Phonetic, w.AudioURL,
		jsonArr(w.Definitions), jsonArr(w.Examples), jsonArr(w.Inflections),
		jsonArr(w.Synonyms), jsonArr(w.Advanced), jsonArr(w.Tags),
		w.Notes, w.ReviewCount,
		fmtTime(w.LastReviewedAt), fmtTime(w.NextReviewAt),
		w.Stability, w.Difficulty, w.State,
		fmtTime(w.CreatedAt), fmtTime(w.UpdatedAt),
	)
	return err
}

func jsonArr(v interface{}) string {
	b, _ := json.Marshal(v)
	if string(b) == "null" {
		return "[]"
	}
	return string(b)
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed") ||
		strings.Contains(err.Error(), "UNIQUE constraint")
}
