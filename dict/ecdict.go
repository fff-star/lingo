package dict

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"lingo/model"

	_ "modernc.org/sqlite"
)

var (
	ecdictDB   *sql.DB
	ecdictOnce sync.Once
	ecdictErr  error
)

// InitECDICT opens the ECDICT SQLite database for read-only queries.
// Returns nil if dbPath is empty (no-op).
func InitECDICT(dbPath string) error {
	ecdictOnce.Do(func() {
		if dbPath == "" {
			return
		}
		db, err := sql.Open("sqlite", dbPath+"?mode=ro")
		if err != nil {
			ecdictErr = fmt.Errorf("open ecdict: %w", err)
			return
		}
		var name string
		if err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='stardict'").Scan(&name); err != nil {
			db.Close()
			ecdictErr = fmt.Errorf("ecdict: stardict table not found")
			return
		}
		ecdictDB = db
	})
	return ecdictErr
}

// ECDICTResult holds the result of an ECDICT lookup.
type ECDICTResult struct {
	Word        string
	Phonetic    string
	Definitions []model.Definition
	Tag         string // space-separated edu tags: cet4, toefl, gre, etc.
}

// LookupECDICT queries the ECDICT database for a word.
func LookupECDICT(word string) (*ECDICTResult, error) {
	if ecdictDB == nil {
		return nil, fmt.Errorf("ecdict not initialized")
	}
	var phonetic, translation, tag string
	err := ecdictDB.QueryRow(
		"SELECT phonetic, translation, tag FROM stardict WHERE word = ? COLLATE NOCASE",
		word,
	).Scan(&phonetic, &translation, &tag)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("word not found in ecdict")
	}
	if err != nil {
		return nil, fmt.Errorf("ecdict query: %w", err)
	}
	return &ECDICTResult{
		Word:        word,
		Phonetic:    cleanECPhonetic(phonetic),
		Definitions: parseTranslation(translation),
		Tag:         tag,
	}, nil
}

// ECDICTEntry is a lightweight result from SearchECDICT prefix search.
type ECDICTEntry struct {
	Word        string             `json:"word"`
	Phonetic    string             `json:"phonetic"`
	Definitions []model.Definition `json:"definitions"`
	Tag         string             `json:"tag"` // space-separated edu tags: cet4, toefl, gre, etc.
}

// SearchECDICT performs a prefix search on the ECDICT database.
// Uses the idx_stardict_word index for fast prefix scans.
func SearchECDICT(prefix string, limit int) ([]ECDICTEntry, error) {
	if ecdictDB == nil {
		return nil, fmt.Errorf("ecdict not initialized")
	}
	if len(prefix) < 2 {
		return nil, nil
	}
	rows, err := ecdictDB.Query(
		"SELECT word, phonetic, translation, tag FROM stardict WHERE word LIKE ? COLLATE NOCASE ORDER BY word LIMIT ?",
		prefix+"%", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("ecdict search: %w", err)
	}
	defer rows.Close()

	var results []ECDICTEntry
	for rows.Next() {
		var word, phonetic, translation, tag string
		if err := rows.Scan(&word, &phonetic, &translation, &tag); err != nil {
			return nil, fmt.Errorf("ecdict scan: %w", err)
		}
		results = append(results, ECDICTEntry{
			Word:        word,
			Phonetic:    cleanECPhonetic(phonetic),
			Definitions: parseTranslation(translation),
			Tag:         tag,
		})
	}
	return results, rows.Err()
}

// parseTranslation parses the ECDICT translation field into Definition structs.
// Lines are "\n"-separated, each optionally prefixed with a POS abbreviation (e.g. "n. text").
func parseTranslation(s string) []model.Definition {
	if s == "" {
		return nil
	}
	// ECDICT data may use literal \n (two chars) or real newlines.
	s = strings.ReplaceAll(s, "\\n", "\n")
	lines := strings.Split(s, "\n")
	defs := make([]model.Definition, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pos := ""
		meaning := line
		// [domain] format, e.g. "[医] 暂时的"
		if strings.HasPrefix(line, "[") {
			if end := strings.Index(line, "]"); end >= 0 {
				pos = line[1:end]
				meaning = strings.TrimSpace(line[end+1:])
			}
		} else if idx := strings.Index(line, ". "); idx >= 0 {
			// POS abbreviation format, e.g. "n. 计算机"
			if isShortPOS(line[:idx]) {
				pos = mapECPOS(line[:idx])
				meaning = strings.TrimSpace(line[idx+2:])
			}
		}
		defs = append(defs, model.Definition{Pos: pos, Meaning: meaning})
	}
	return defs
}

// mapECPOS maps ECDICT POS abbreviations to full forms.
func mapECPOS(s string) string {
	switch s {
	case "a":
		return "adj"
	default:
		return s
	}
}

func isShortPOS(s string) bool {
	switch s {
	case "a", "n", "v", "adj", "adv", "pron", "prep", "conj", "int",
		"art", "vt", "vi", "aux", "abbr", "pl", "num", "det":
		return true
	}
	return false
}

// CloseECDICT closes the ECDICT database connection.
func CloseECDICT() {
	if ecdictDB != nil {
		ecdictDB.Close()
		ecdictDB = nil
	}
}

// cleanECPhonetic normalises ECDICT phonetic notation.
func cleanECPhonetic(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s != "" {
		return "/" + s + "/"
	}
	return ""
}
