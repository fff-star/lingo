package model

import "time"

type Definition struct {
	Pos     string `json:"pos"`
	Meaning string `json:"meaning"`
}

type Inflection struct {
	Form  string `json:"form"`
	Value string `json:"value"`
}

type Word struct {
	ID             string       `json:"id"`
	Word           string       `json:"word"`
	Phonetic       string       `json:"phonetic"`
	Definitions    []Definition `json:"definitions"`
	Examples       []string     `json:"examples"`
	Inflections    []Inflection `json:"inflections"`
	Synonyms       []string     `json:"synonyms"`
	Advanced       []string     `json:"advanced"`
	Tags           []string     `json:"tags"`
	Notes          string       `json:"notes"`
	ReviewCount    int       `json:"review_count"`
	LastReviewedAt time.Time `json:"last_reviewed_at"`
	NextReviewAt   time.Time `json:"next_review_at"`
	Stability      float64   `json:"stability"`
	Difficulty     float64   `json:"difficulty"`
	State          int       `json:"state"` // 0=New, 1=Learning, 2=Review, 3=Relearning
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}
