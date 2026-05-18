package model

import "time"

type Sentence struct {
	ID          string    `json:"id"`
	Text        string    `json:"text"`
	Source      string    `json:"source"`
	SourceURL   string    `json:"source_url"`
	Author      string    `json:"author"`
	Translation string    `json:"translation"`
	Tags        []string  `json:"tags"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
