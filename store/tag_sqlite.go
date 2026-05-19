package store

import (
	"database/sql"

	"lingo/model"
)

type sqliteTagStore struct {
	db *sql.DB
}

func NewTagStore(db *sql.DB) TagStore {
	return &sqliteTagStore{db: db}
}

func (s *sqliteTagStore) Load() ([]model.Tag, error) {
	rows, err := s.db.Query("SELECT id, name, color, description, category FROM tags ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.Description, &t.Category); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *sqliteTagStore) Save(tags []model.Tag) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM tags"); err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO tags (id, name, color, description, category) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range tags {
		if _, err := stmt.Exec(t.ID, t.Name, t.Color, t.Description, t.Category); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *sqliteTagStore) Add(tag model.Tag) error {
	_, err := s.db.Exec("INSERT INTO tags (id, name, color, description, category) VALUES (?, ?, ?, ?, ?)",
		tag.ID, tag.Name, tag.Color, tag.Description, tag.Category)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrExists
		}
		return err
	}
	return nil
}

func (s *sqliteTagStore) Get(name string) (*model.Tag, error) {
	t, err := scanTag(s.db.QueryRow("SELECT id, name, color, description, category FROM tags WHERE name = ? OR id = ?",
		name, name))
	if err == nil {
		return &t, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	rows, err := s.db.Query("SELECT id, name, color, description, category FROM tags WHERE name LIKE ? OR id LIKE ? LIMIT 1",
		name+"%", name+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.Description, &t.Category); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	if len(tags) == 0 {
		return nil, ErrNotFound
	}
	return &tags[0], rows.Err()
}

func (s *sqliteTagStore) Rename(oldName, newName string) error {
	result, err := s.db.Exec("UPDATE tags SET name = ? WHERE name = ?", newName, oldName)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrExists
		}
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteTagStore) Delete(name string) error {
	result, err := s.db.Exec("DELETE FROM tags WHERE name = ? OR id LIKE ?", name, name+"%")
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanTag(row interface{ Scan(...interface{}) error }) (model.Tag, error) {
	var t model.Tag
	err := row.Scan(&t.ID, &t.Name, &t.Color, &t.Description, &t.Category)
	return t, err
}
