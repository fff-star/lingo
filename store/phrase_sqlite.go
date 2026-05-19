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

type sqlitePhraseStore struct {
	db *sql.DB
}

func NewPhraseStore(db *sql.DB) PhraseStore {
	return &sqlitePhraseStore{db: db}
}

func (s *sqlitePhraseStore) Load() ([]model.Phrase, error) {
	rows, err := s.db.Query(`SELECT id, phrase, type, words, definition, examples,
		synonyms, advanced, tags, notes, review_count,
		last_reviewed_at, next_review_at, stability, difficulty, state,
		created_at, updated_at FROM phrases ORDER BY phrase`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPhrases(rows)
}

func (s *sqlitePhraseStore) LoadDue() ([]model.Phrase, error) {
	now := fmtTime(time.Now().UTC())
	rows, err := s.db.Query(`SELECT id, phrase, type, words, definition, examples,
		synonyms, advanced, tags, notes, review_count,
		last_reviewed_at, next_review_at, stability, difficulty, state,
		created_at, updated_at FROM phrases
		WHERE next_review_at <= ? OR next_review_at = '0001-01-01T00:00:00Z'
		ORDER BY next_review_at`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPhrases(rows)
}

func (s *sqlitePhraseStore) Add(p model.Phrase) error {
	_, err := s.db.Exec(`INSERT INTO phrases
		(id, phrase, type, words, definition, examples, synonyms, advanced, tags, notes,
		 review_count, last_reviewed_at, next_review_at, stability, difficulty, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Phrase, p.Type, jsonArr(p.Words), p.Definition,
		jsonArr(p.Examples), jsonArr(p.Synonyms), jsonArr(p.Advanced), jsonArr(p.Tags),
		p.Notes, p.ReviewCount,
		fmtTime(p.LastReviewedAt), fmtTime(p.NextReviewAt),
		p.Stability, p.Difficulty, p.State,
		fmtTime(p.CreatedAt), fmtTime(p.UpdatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrExists
		}
		return err
	}
	return nil
}

func (s *sqlitePhraseStore) Get(idPrefix string) (*model.Phrase, error) {
	p, err := scanPhrase(s.db.QueryRow(`SELECT id, phrase, type, words, definition, examples,
		synonyms, advanced, tags, notes, review_count,
		last_reviewed_at, next_review_at, stability, difficulty, state,
		created_at, updated_at FROM phrases WHERE id = ? OR phrase = ?`,
		idPrefix, idPrefix))
	if err == nil {
		return &p, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	rows, err := s.db.Query(`SELECT id, phrase, type, words, definition, examples,
		synonyms, advanced, tags, notes, review_count,
		last_reviewed_at, next_review_at, stability, difficulty, state,
		created_at, updated_at FROM phrases WHERE id LIKE ? OR phrase LIKE ? LIMIT 1`,
		idPrefix+"%", idPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	phrases, err := scanPhrases(rows)
	if err != nil {
		return nil, err
	}
	if len(phrases) == 0 {
		return nil, ErrNotFound
	}
	return &phrases[0], nil
}

func (s *sqlitePhraseStore) Update(p model.Phrase) error {
	result, err := s.db.Exec(`UPDATE phrases SET phrase=?, type=?, words=?, definition=?, examples=?,
		synonyms=?, advanced=?, tags=?, notes=?, review_count=?,
		last_reviewed_at=?, next_review_at=?, stability=?, difficulty=?, state=?,
		updated_at=? WHERE id=?`,
		p.Phrase, p.Type, jsonArr(p.Words), p.Definition,
		jsonArr(p.Examples), jsonArr(p.Synonyms), jsonArr(p.Advanced), jsonArr(p.Tags),
		p.Notes, p.ReviewCount,
		fmtTime(p.LastReviewedAt), fmtTime(p.NextReviewAt),
		p.Stability, p.Difficulty, p.State,
		fmtTime(p.UpdatedAt),
		p.ID,
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

func (s *sqlitePhraseStore) Delete(idPrefix string) error {
	result, err := s.db.Exec(`DELETE FROM phrases WHERE id LIKE ? OR phrase = ?`,
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

func (s *sqlitePhraseStore) Search(keywords []string, tags []string) ([]model.Phrase, error) {
	if len(keywords) > 0 {
		if results, ok := s.ftsSearch(keywords, tags); ok {
			return results, nil
		}
	}

	query := `SELECT id, phrase, type, words, definition, examples,
		synonyms, advanced, tags, notes, review_count,
		last_reviewed_at, next_review_at, stability, difficulty, state,
		created_at, updated_at FROM phrases WHERE 1=1`
	var args []interface{}

	for _, kw := range keywords {
		kw = "%" + strings.ToLower(kw) + "%"
		query += ` AND (LOWER(phrase) LIKE ? OR LOWER(definition) LIKE ? OR LOWER(synonyms) LIKE ? OR LOWER(advanced) LIKE ?)`
		args = append(args, kw, kw, kw, kw)
	}
	for _, tag := range tags {
		query += ` AND tags LIKE ?`
		args = append(args, fmt.Sprintf(`%%"%s"%%`, tag))
	}
	query += ` ORDER BY phrase`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPhrases(rows)
}

func (s *sqlitePhraseStore) ftsSearch(keywords, tags []string) ([]model.Phrase, bool) {
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

	sql := `SELECT p.id, p.phrase, p.type, p.words, p.definition, p.examples,
		p.synonyms, p.advanced, p.tags, p.notes, p.review_count,
		p.last_reviewed_at, p.next_review_at, p.stability, p.difficulty, p.state,
		p.created_at, p.updated_at
		FROM phrases p
		INNER JOIN phrases_fts f ON p.rowid = f.rowid
		WHERE phrases_fts MATCH ?`
	var args []interface{}
	args = append(args, ftsQuery)

	for _, tag := range tags {
		sql += ` AND p.tags LIKE ?`
		args = append(args, fmt.Sprintf(`%%"%s"%%`, tag))
	}
	sql += ` ORDER BY rank`

	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	results, err := scanPhrases(rows)
	if err != nil || len(results) == 0 {
		return nil, false
	}
	return results, true
}

func (s *sqlitePhraseStore) GetAllTags() ([]string, error) {
	phrases, err := s.Load()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var tags []string
	for _, p := range phrases {
		for _, t := range p.Tags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	sort.Strings(tags)
	return tags, nil
}

func (s *sqlitePhraseStore) Count() (int, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(*) FROM phrases").Scan(&n)
	return n, err
}

func scanPhrase(row interface{ Scan(...interface{}) error }) (model.Phrase, error) {
	var p model.Phrase
	var wds, exs, syns, advs, tgs string
	var lra, nra, ca, ua string
	err := row.Scan(
		&p.ID, &p.Phrase, &p.Type,
		&wds, &p.Definition, &exs, &syns, &advs, &tgs,
		&p.Notes,
		&p.ReviewCount, &lra, &nra,
		&p.Stability, &p.Difficulty, &p.State,
		&ca, &ua,
	)
	if err != nil {
		return p, err
	}
	json.Unmarshal([]byte(wds), &p.Words)
	json.Unmarshal([]byte(exs), &p.Examples)
	json.Unmarshal([]byte(syns), &p.Synonyms)
	json.Unmarshal([]byte(advs), &p.Advanced)
	json.Unmarshal([]byte(tgs), &p.Tags)
	p.LastReviewedAt = parseTime(lra)
	p.NextReviewAt = parseTime(nra)
	p.CreatedAt = parseTime(ca)
	p.UpdatedAt = parseTime(ua)
	return p, nil
}

func scanPhrases(rows *sql.Rows) ([]model.Phrase, error) {
	var phrases []model.Phrase
	for rows.Next() {
		p, err := scanPhrase(rows)
		if err != nil {
			return nil, err
		}
		phrases = append(phrases, p)
	}
	return phrases, rows.Err()
}
