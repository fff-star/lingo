package model

import "time"

type Article struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Author     string      `json:"author"`
	Source     string      `json:"source"`
	SourceURL  string      `json:"source_url"`
	Content    string      `json:"content"`
	Summary    string      `json:"summary"`
	Tags       []string    `json:"tags"`
	Notes      string      `json:"notes"`
	AIAnalysis *AIAnalysis `json:"ai_analysis,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}
