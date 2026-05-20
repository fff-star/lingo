package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrNotFound = errors.New("not found")
	ErrExists   = errors.New("already exists")
)

func NewID(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand.Read failed: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}

func EnsureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

func readJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func writeJSON(path string, v interface{}) error {
	if err := EnsureDir(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func matchAll(s string, keywords []string) bool {
	for _, kw := range keywords {
		if !strings.Contains(strings.ToLower(s), strings.ToLower(kw)) {
			return false
		}
	}
	return true
}

func HasString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, target string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != target {
			result = append(result, s)
		}
	}
	return result
}

func MatchAnyTag(tags []string, filterTags []string) bool {
	if len(filterTags) == 0 {
		return true
	}
	for _, ft := range filterTags {
		for _, t := range tags {
			if t == ft {
				return true
			}
		}
	}
	return false
}
