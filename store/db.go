package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// OpenDB opens the SQLite database and ensures tables exist.
func OpenDB(dbPath string) (*sql.DB, error) {
	if err := EnsureDir(dbPath); err != nil {
		return nil, fmt.Errorf("ensure db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// single conn avoids SQLITE_BUSY from concurrent writes in embedded mode.
	db.SetMaxOpenConns(1)

	if err := CreateTables(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("create tables: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return db, nil
}

func CreateTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS words (
		id TEXT PRIMARY KEY,
		word TEXT NOT NULL COLLATE NOCASE,
		phonetic TEXT NOT NULL DEFAULT '',
		definitions TEXT NOT NULL DEFAULT '[]',
		ecdict_defs TEXT NOT NULL DEFAULT '[]',
		examples TEXT NOT NULL DEFAULT '[]',
		inflections TEXT NOT NULL DEFAULT '[]',
		synonyms TEXT NOT NULL DEFAULT '[]',
		advanced TEXT NOT NULL DEFAULT '[]',
		tags TEXT NOT NULL DEFAULT '[]',
		notes TEXT NOT NULL DEFAULT '',
		audio_url TEXT NOT NULL DEFAULT '',
		review_count INTEGER NOT NULL DEFAULT 0,
		last_reviewed_at TEXT NOT NULL DEFAULT '0001-01-01T00:00:00Z',
		next_review_at TEXT NOT NULL DEFAULT '0001-01-01T00:00:00Z',
		stability REAL NOT NULL DEFAULT 0,
		difficulty REAL NOT NULL DEFAULT 5.0,
		state INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_words_word ON words(word COLLATE NOCASE);
	CREATE INDEX IF NOT EXISTS idx_words_next_review ON words(next_review_at);
	CREATE INDEX IF NOT EXISTS idx_words_created ON words(created_at);

	CREATE TABLE IF NOT EXISTS phrases (
		id TEXT PRIMARY KEY,
		phrase TEXT NOT NULL COLLATE NOCASE,
		type TEXT NOT NULL DEFAULT '',
		words TEXT NOT NULL DEFAULT '[]',
		definition TEXT NOT NULL DEFAULT '',
		examples TEXT NOT NULL DEFAULT '[]',
		synonyms TEXT NOT NULL DEFAULT '[]',
		advanced TEXT NOT NULL DEFAULT '[]',
		tags TEXT NOT NULL DEFAULT '[]',
		notes TEXT NOT NULL DEFAULT '',
		review_count INTEGER NOT NULL DEFAULT 0,
		last_reviewed_at TEXT NOT NULL DEFAULT '0001-01-01T00:00:00Z',
		next_review_at TEXT NOT NULL DEFAULT '0001-01-01T00:00:00Z',
		stability REAL NOT NULL DEFAULT 0,
		difficulty REAL NOT NULL DEFAULT 5.0,
		state INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_phrases_phrase ON phrases(phrase COLLATE NOCASE);
	CREATE INDEX IF NOT EXISTS idx_phrases_next_review ON phrases(next_review_at);
	CREATE INDEX IF NOT EXISTS idx_phrases_created ON phrases(created_at);

	CREATE TABLE IF NOT EXISTS sentences (
		id TEXT PRIMARY KEY,
		text TEXT NOT NULL,
		source TEXT NOT NULL DEFAULT '',
		source_url TEXT NOT NULL DEFAULT '',
		author TEXT NOT NULL DEFAULT '',
		translation TEXT NOT NULL DEFAULT '',
		tags TEXT NOT NULL DEFAULT '[]',
		notes TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_sentences_created ON sentences(created_at);

	CREATE TABLE IF NOT EXISTS articles (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL DEFAULT '',
		source_url TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL DEFAULT '',
		summary TEXT NOT NULL DEFAULT '',
		tags TEXT NOT NULL DEFAULT '[]',
		notes TEXT NOT NULL DEFAULT '',
		ai_analysis TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_articles_created ON articles(created_at);

	CREATE TABLE IF NOT EXISTS compositions (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL DEFAULT '',
		tags TEXT NOT NULL DEFAULT '[]',
		notes TEXT NOT NULL DEFAULT '',
		ai_analysis TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_compositions_created ON compositions(created_at);

	CREATE TABLE IF NOT EXISTS tags (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL COLLATE NOCASE,
		color TEXT NOT NULL DEFAULT '#888888',
		description TEXT NOT NULL DEFAULT '',
		category TEXT NOT NULL DEFAULT ''
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name ON tags(name COLLATE NOCASE);

	CREATE TABLE IF NOT EXISTS review_log (
		date TEXT PRIMARY KEY,
		count INTEGER NOT NULL DEFAULT 0
	);

	-- FTS5 virtual tables for full-text search
	CREATE VIRTUAL TABLE IF NOT EXISTS words_fts USING fts5(
		word, definitions, ecdict_defs, synonyms, advanced, notes,
		content=words, content_rowid=rowid
	);
	CREATE VIRTUAL TABLE IF NOT EXISTS phrases_fts USING fts5(
		phrase, definition, synonyms, advanced, notes,
		content=phrases, content_rowid=rowid
	);
	CREATE VIRTUAL TABLE IF NOT EXISTS sentences_fts USING fts5(
		text, translation, source, notes,
		content=sentences, content_rowid=rowid
	);
	CREATE VIRTUAL TABLE IF NOT EXISTS articles_fts USING fts5(
		title, content, summary, notes,
		content=articles, content_rowid=rowid
	);
	CREATE VIRTUAL TABLE IF NOT EXISTS compositions_fts USING fts5(
		title, content, notes,
		content=compositions, content_rowid=rowid
	);

	CREATE TABLE IF NOT EXISTS schema_version (version INTEGER);

	-- FTS sync triggers for words
	DROP TRIGGER IF EXISTS words_fts_insert;
	CREATE TRIGGER words_fts_insert AFTER INSERT ON words BEGIN
		INSERT INTO words_fts(rowid, word, definitions, ecdict_defs, synonyms, advanced, notes)
		VALUES (new.rowid, new.word, new.definitions, new.ecdict_defs, new.synonyms, new.advanced, new.notes);
	END;

	DROP TRIGGER IF EXISTS words_fts_delete;
	CREATE TRIGGER words_fts_delete AFTER DELETE ON words BEGIN
		INSERT INTO words_fts(words_fts, rowid, word, definitions, ecdict_defs, synonyms, advanced, notes)
		VALUES ('delete', old.rowid, old.word, old.definitions, old.ecdict_defs, old.synonyms, old.advanced, old.notes);
	END;

	DROP TRIGGER IF EXISTS words_fts_update;
	CREATE TRIGGER words_fts_update AFTER UPDATE ON words BEGIN
		INSERT INTO words_fts(words_fts, rowid, word, definitions, ecdict_defs, synonyms, advanced, notes)
		VALUES ('delete', old.rowid, old.word, old.definitions, old.ecdict_defs, old.synonyms, old.advanced, old.notes);
		INSERT INTO words_fts(rowid, word, definitions, ecdict_defs, synonyms, advanced, notes)
		VALUES (new.rowid, new.word, new.definitions, new.ecdict_defs, new.synonyms, new.advanced, new.notes);
	END;

	-- FTS sync triggers for phrases
	DROP TRIGGER IF EXISTS phrases_fts_insert;
	CREATE TRIGGER phrases_fts_insert AFTER INSERT ON phrases BEGIN
		INSERT INTO phrases_fts(rowid, phrase, definition, synonyms, advanced, notes)
		VALUES (new.rowid, new.phrase, new.definition, new.synonyms, new.advanced, new.notes);
	END;

	DROP TRIGGER IF EXISTS phrases_fts_delete;
	CREATE TRIGGER phrases_fts_delete AFTER DELETE ON phrases BEGIN
		INSERT INTO phrases_fts(phrases_fts, rowid, phrase, definition, synonyms, advanced, notes)
		VALUES ('delete', old.rowid, old.phrase, old.definition, old.synonyms, old.advanced, old.notes);
	END;

	DROP TRIGGER IF EXISTS phrases_fts_update;
	CREATE TRIGGER phrases_fts_update AFTER UPDATE ON phrases BEGIN
		INSERT INTO phrases_fts(phrases_fts, rowid, phrase, definition, synonyms, advanced, notes)
		VALUES ('delete', old.rowid, old.phrase, old.definition, old.synonyms, old.advanced, old.notes);
		INSERT INTO phrases_fts(rowid, phrase, definition, synonyms, advanced, notes)
		VALUES (new.rowid, new.phrase, new.definition, new.synonyms, new.advanced, new.notes);
	END;

	-- FTS sync triggers for sentences
	DROP TRIGGER IF EXISTS sentences_fts_insert;
	CREATE TRIGGER sentences_fts_insert AFTER INSERT ON sentences BEGIN
		INSERT INTO sentences_fts(rowid, text, translation, source, notes)
		VALUES (new.rowid, new.text, new.translation, new.source, new.notes);
	END;

	DROP TRIGGER IF EXISTS sentences_fts_delete;
	CREATE TRIGGER sentences_fts_delete AFTER DELETE ON sentences BEGIN
		INSERT INTO sentences_fts(sentences_fts, rowid, text, translation, source, notes)
		VALUES ('delete', old.rowid, old.text, old.translation, old.source, old.notes);
	END;

	DROP TRIGGER IF EXISTS sentences_fts_update;
	CREATE TRIGGER sentences_fts_update AFTER UPDATE ON sentences BEGIN
		INSERT INTO sentences_fts(sentences_fts, rowid, text, translation, source, notes)
		VALUES ('delete', old.rowid, old.text, old.translation, old.source, old.notes);
		INSERT INTO sentences_fts(rowid, text, translation, source, notes)
		VALUES (new.rowid, new.text, new.translation, new.source, new.notes);
	END;

	-- FTS sync triggers for articles
	DROP TRIGGER IF EXISTS articles_fts_insert;
	CREATE TRIGGER articles_fts_insert AFTER INSERT ON articles BEGIN
		INSERT INTO articles_fts(rowid, title, content, summary, notes)
		VALUES (new.rowid, new.title, new.content, new.summary, new.notes);
	END;

	DROP TRIGGER IF EXISTS articles_fts_delete;
	CREATE TRIGGER articles_fts_delete AFTER DELETE ON articles BEGIN
		INSERT INTO articles_fts(articles_fts, rowid, title, content, summary, notes)
		VALUES ('delete', old.rowid, old.title, old.content, old.summary, old.notes);
	END;

	DROP TRIGGER IF EXISTS articles_fts_update;
	CREATE TRIGGER articles_fts_update AFTER UPDATE ON articles BEGIN
		INSERT INTO articles_fts(articles_fts, rowid, title, content, summary, notes)
		VALUES ('delete', old.rowid, old.title, old.content, old.summary, old.notes);
		INSERT INTO articles_fts(rowid, title, content, summary, notes)
		VALUES (new.rowid, new.title, new.content, new.summary, new.notes);
	END;

	-- FTS sync triggers for compositions
	DROP TRIGGER IF EXISTS compositions_fts_insert;
	CREATE TRIGGER compositions_fts_insert AFTER INSERT ON compositions BEGIN
		INSERT INTO compositions_fts(rowid, title, content, notes)
		VALUES (new.rowid, new.title, new.content, new.notes);
	END;

	DROP TRIGGER IF EXISTS compositions_fts_delete;
	CREATE TRIGGER compositions_fts_delete AFTER DELETE ON compositions BEGIN
		INSERT INTO compositions_fts(compositions_fts, rowid, title, content, notes)
		VALUES ('delete', old.rowid, old.title, old.content, old.notes);
	END;

	DROP TRIGGER IF EXISTS compositions_fts_update;
	CREATE TRIGGER compositions_fts_update AFTER UPDATE ON compositions BEGIN
		INSERT INTO compositions_fts(compositions_fts, rowid, title, content, notes)
		VALUES ('delete', old.rowid, old.title, old.content, old.notes);
		INSERT INTO compositions_fts(rowid, title, content, notes)
		VALUES (new.rowid, new.title, new.content, new.notes);
	END;
	`

	_, err := db.Exec(schema)
	return err
}

func runMigrations(db *sql.DB) error {
	// Get current schema version.
	var version int
	if err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version); err != nil {
		// DB inconsistency (table missing after createTables) — start fresh at v0.
		version = 0
	}

	if version < 1 {
		if err := migrateV1(db); err != nil {
			return fmt.Errorf("migration v1: %w", err)
		}
	}
	if version < 2 {
		if err := migrateV2(db); err != nil {
			return fmt.Errorf("migration v2: %w", err)
		}
	}
	if version < 3 {
		if err := migrateV3(db); err != nil {
			return fmt.Errorf("migration v3: %w", err)
		}
	}

	return nil
}

func migrateV1(db *sql.DB) error {
	// Add audio_url column if it doesn't exist.
	cols, err := db.Query("PRAGMA table_info(words)")
	if err != nil {
		return fmt.Errorf("migration v1: %w", err)
	}
	hasAudioURL := false
	for cols.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultVal *string
		var pk int
		cols.Scan(&cid, &name, &colType, &notNull, &defaultVal, &pk)
		if name == "audio_url" {
			hasAudioURL = true
			break
		}
	}
	cols.Close() // must close before subsequent queries to avoid lock
	if !hasAudioURL {
		if _, err := db.Exec("ALTER TABLE words ADD COLUMN audio_url TEXT NOT NULL DEFAULT ''"); err != nil {
			return fmt.Errorf("add audio_url column: %w", err)
		}
	}

	// Rebuild all FTS indexes from content tables.
	for _, table := range []string{"words", "phrases", "sentences", "articles", "compositions"} {
		fts := table + "_fts"
		if _, err := db.Exec(fmt.Sprintf("INSERT INTO %s(%s) VALUES('rebuild')", fts, fts)); err != nil {
			return fmt.Errorf("rebuild %s: %w", fts, err)
		}
	}

	if _, err := db.Exec("INSERT INTO schema_version(version) VALUES (1)"); err != nil {
		return fmt.Errorf("record v1: %w", err)
	}
	return nil
}

func migrateV2(db *sql.DB) error {
	// Fix difficulty default: cards created before the schema fix have difficulty=0.
	// FSRS requires initial difficulty = w[4] = 5.0.
	if _, err := db.Exec("UPDATE words SET difficulty = 5.0 WHERE difficulty = 0"); err != nil {
		return fmt.Errorf("fix words difficulty: %w", err)
	}
	if _, err := db.Exec("UPDATE phrases SET difficulty = 5.0 WHERE difficulty = 0"); err != nil {
		return fmt.Errorf("fix phrases difficulty: %w", err)
	}
	if _, err := db.Exec("INSERT INTO schema_version(version) VALUES (2)"); err != nil {
		return fmt.Errorf("record v2: %w", err)
	}
	return nil
}

func migrateV3(db *sql.DB) error {
	// Add ecdict_defs column if it doesn't exist.
	cols, err := db.Query("PRAGMA table_info(words)")
	if err != nil {
		return fmt.Errorf("migration v3: %w", err)
	}
	hasECDict := false
	for cols.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultVal *string
		var pk int
		cols.Scan(&cid, &name, &colType, &notNull, &defaultVal, &pk)
		if name == "ecdict_defs" {
			hasECDict = true
			break
		}
	}
	cols.Close()
	if !hasECDict {
		if _, err := db.Exec("ALTER TABLE words ADD COLUMN ecdict_defs TEXT NOT NULL DEFAULT '[]'"); err != nil {
			return fmt.Errorf("add ecdict_defs: %w", err)
		}
	}
	// Recreate FTS to include the new ecdict_defs column.
	if _, err := db.Exec(`
		DROP TRIGGER IF EXISTS words_fts_insert;
		DROP TRIGGER IF EXISTS words_fts_delete;
		DROP TRIGGER IF EXISTS words_fts_update;
		DROP TABLE IF EXISTS words_fts;
		CREATE VIRTUAL TABLE words_fts USING fts5(
			word, definitions, ecdict_defs, synonyms, advanced, notes,
			content=words, content_rowid=rowid
		);
		CREATE TRIGGER words_fts_insert AFTER INSERT ON words BEGIN
			INSERT INTO words_fts(rowid, word, definitions, ecdict_defs, synonyms, advanced, notes)
			VALUES (new.rowid, new.word, new.definitions, new.ecdict_defs, new.synonyms, new.advanced, new.notes);
		END;
		CREATE TRIGGER words_fts_delete AFTER DELETE ON words BEGIN
			INSERT INTO words_fts(words_fts, rowid, word, definitions, ecdict_defs, synonyms, advanced, notes)
			VALUES ('delete', old.rowid, old.word, old.definitions, old.ecdict_defs, old.synonyms, old.advanced, old.notes);
		END;
		CREATE TRIGGER words_fts_update AFTER UPDATE ON words BEGIN
			INSERT INTO words_fts(words_fts, rowid, word, definitions, ecdict_defs, synonyms, advanced, notes)
			VALUES ('delete', old.rowid, old.word, old.definitions, old.ecdict_defs, old.synonyms, old.advanced, old.notes);
			INSERT INTO words_fts(rowid, word, definitions, ecdict_defs, synonyms, advanced, notes)
			VALUES (new.rowid, new.word, new.definitions, new.ecdict_defs, new.synonyms, new.advanced, new.notes);
		END;
	`); err != nil {
		return fmt.Errorf("rebuild words_fts v3: %w", err)
	}
	// Repopulate FTS from content table.
	if _, err := db.Exec("INSERT INTO words_fts(words_fts) VALUES('rebuild')"); err != nil {
		return fmt.Errorf("repopulate words_fts v3: %w", err)
	}
	if _, err := db.Exec("INSERT INTO schema_version(version) VALUES (3)"); err != nil {
		return fmt.Errorf("record v3: %w", err)
	}
	return nil
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return "0001-01-01T00:00:00Z"
	}
	return t.Format(time.RFC3339Nano)
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}
