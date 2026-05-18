package model

import "time"

type Phrase struct {
	ID             string    `json:"id"`
	Phrase         string    `json:"phrase"`
	Type           string    `json:"type"`
	Words          []string  `json:"words"`
	Definition     string    `json:"definition"`
	Examples       []string  `json:"examples"`
	Synonyms       []string  `json:"synonyms"`
	Advanced       []string  `json:"advanced"`
	Tags           []string  `json:"tags"`
	Notes          string    `json:"notes"`
	ReviewCount    int       `json:"review_count"`
	LastReviewedAt time.Time `json:"last_reviewed_at"`
	NextReviewAt   time.Time `json:"next_review_at"`
	Stability      float64   `json:"stability"`
	Difficulty     float64   `json:"difficulty"`
	State          int       `json:"state"` // 0=New, 1=Learning, 2=Review, 3=Relearning
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
