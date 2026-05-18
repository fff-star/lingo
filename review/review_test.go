package review

import (
	"math"
	"testing"
	"time"
)

func TestNewCardState(t *testing.T) {
	s, d, state := NewCardState(DefaultWeights)
	if s != 0 {
		t.Errorf("new card stability = %f, want 0", s)
	}
	if d != DefaultWeights[4] {
		t.Errorf("new card difficulty = %f, want %f", d, DefaultWeights[4])
	}
	if state != StateNew {
		t.Errorf("new card state = %d, want %d", state, StateNew)
	}
}

func TestFirstReview(t *testing.T) {
	tests := []struct {
		rating    int
		wantState int
	}{
		{RatingAgain, StateReview},
		{RatingHard, StateReview},
		{RatingGood, StateReview},
		{RatingEasy, StateReview},
	}

	for _, tt := range tests {
		s, d, state := NewCardState(DefaultWeights)
		newS, newD, newState, days := Review(s, d, state, 0, tt.rating, DefaultWeights)

		if newState != tt.wantState {
			t.Errorf("rating=%d: state = %d, want %d", tt.rating, newState, tt.wantState)
		}
		if newS <= 0 {
			t.Errorf("rating=%d: stability = %f, want > 0", tt.rating, newS)
		}
		if newD < 1 || newD > 10 {
			t.Errorf("rating=%d: difficulty = %f, want in [1,10]", tt.rating, newD)
		}
		if days < 1 {
			t.Errorf("rating=%d: scheduled days = %d, want >= 1", tt.rating, days)
		}
	}
}

func TestReviewStabilityIncrease(t *testing.T) {
	s, d, state := NewCardState(DefaultWeights)

	// First review: Good.
	s, d, state, _ = Review(s, d, state, 0, RatingGood, DefaultWeights)
	if state != StateReview {
		t.Fatalf("state after first review = %d, want %d", state, StateReview)
	}
	s1 := s

	// Second review after exact interval: Good again, stability should increase.
	s, d, state, days := Review(s, d, state, s1, RatingGood, DefaultWeights)
	if s <= s1 {
		t.Errorf("stability after second Good: %f <= %f, expected increase", s, s1)
	}
	if days < 1 {
		t.Errorf("days after second Good: %d, want >= 1", days)
	}
}

func TestReviewAgainResets(t *testing.T) {
	s, d, state := NewCardState(DefaultWeights)

	// First review: Good.
	s, d, state, _ = Review(s, d, state, 0, RatingGood, DefaultWeights)

	// Second review: Again.
	s, d, state, _ = Review(s, d, state, s, RatingAgain, DefaultWeights)
	if state != StateRelearning {
		t.Errorf("state after Again = %d, want %d (Relearning)", state, StateRelearning)
	}
	// Stability after failure should be lower.
	if s > DefaultWeights[2]*2 {
		t.Errorf("stability after failure = %f, expected reset to low value", s)
	}
}

func TestDifficultyBounds(t *testing.T) {
	s, d, state := NewCardState(DefaultWeights)

	// Keep rating Easy many times to push difficulty down.
	for i := 0; i < 20; i++ {
		s, d, state, _ = Review(s, d, state, s, RatingEasy, DefaultWeights)
		if state == StateRelearning {
			// Promote back to review.
			state = StateReview
		}
	}
	if d < 1 {
		t.Errorf("difficulty dropped below 1: %f", d)
	}
	if d > 10 {
		t.Errorf("difficulty exceeded 10: %f", d)
	}

	// Keep rating Again to push difficulty up.
	s, d, state = NewCardState(DefaultWeights)
	for i := 0; i < 20; i++ {
		s, d, state, _ = Review(s, d, state, s, RatingAgain, DefaultWeights)
		if state == StateRelearning {
			state = StateReview
		}
	}
	if d < 1 || d > 10 {
		t.Errorf("difficulty out of bounds after many Again: %f", d)
	}
}

func TestDue(t *testing.T) {
	past := time.Now().UTC().Add(-1 * time.Hour)
	future := time.Now().UTC().Add(24 * time.Hour)

	if !Due(past) {
		t.Error("past time should be due")
	}
	if Due(future) {
		t.Error("future time should not be due")
	}
}

func TestDaysLate(t *testing.T) {
	past := time.Now().UTC().Add(-48 * time.Hour)
	days := DaysLate(past)
	if math.Abs(days-2) > 0.1 {
		t.Errorf("48h late = %f days, want ~2", days)
	}
}

func TestRetrievability(t *testing.T) {
	// After exactly stability days, retrievability should be 0.9.
	r := retrievability(10, 10)
	if math.Abs(r-0.9) > 0.001 {
		t.Errorf("retrievability(10, 10) = %f, want 0.9", r)
	}

	// No memory — retrievability should be 1.
	r = retrievability(5, 0)
	if r != 1 {
		t.Errorf("retrievability with 0 stability = %f, want 1", r)
	}
}

func TestNextDayStart(t *testing.T) {
	// Simulate a review at 10:30 AM on a known date.
	loc := time.UTC
	base := time.Date(2026, 5, 18, 10, 30, 0, 0, loc)

	// 1 day → next day 00:00.
	next := NextDayStart(base, 1)
	expected := time.Date(2026, 5, 19, 0, 0, 0, 0, loc)
	if !next.Equal(expected) {
		t.Errorf("10:30 + 1 day = %s, want %s", next, expected)
	}

	// 0 days → today 00:00 (but after base, so tomorrow 00:00).
	next = NextDayStart(base, 0)
	if !next.After(base) {
		t.Errorf("0-day result must be after base time: got %s", next)
	}

	// Midnight grace period: 00:30 should be treated as previous day.
	midnight := time.Date(2026, 5, 19, 0, 30, 0, 0, loc)
	next = NextDayStart(midnight, 1)
	// Should be May 19 00:00 + 1 = May 20 00:00.
	expected = time.Date(2026, 5, 20, 0, 0, 0, 0, loc)
	if !next.Equal(expected) {
		t.Errorf("00:30 + 1 day = %s, want %s", next, expected)
	}

	// Large interval is capped.
	next = NextDayStart(base, 36500)
	if next.Before(base) {
		t.Error("large interval should be in the future")
	}
}

func TestNextInterval(t *testing.T) {
	days := NextInterval(10, 0.9)
	if days < 1 {
		t.Errorf("NextInterval(10, 0.9) = %d, want >= 1", days)
	}
	if days > 36500 {
		t.Errorf("NextInterval(10, 0.9) = %d, want <= 36500", days)
	}
}
