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
		wantDays  bool // true = expect days >= 1, false = expect days == 0
	}{
		{RatingAgain, StateLearning, false},
		{RatingHard, StateLearning, false},
		{RatingGood, StateReview, true},
		{RatingEasy, StateReview, true},
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
		if tt.wantDays && days < 1 {
			t.Errorf("rating=%d: scheduled days = %d, want >= 1", tt.rating, days)
		}
		if !tt.wantDays && days != 0 {
			t.Errorf("rating=%d: scheduled days = %d, want 0", tt.rating, days)
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

// ── Learning & Relearning state machine boundary tests ──

func TestLearningStayOnAgainHard(t *testing.T) {
	// A Learning card rated Again or Hard stays in Learning with 0 days.
	for _, rating := range []int{RatingAgain, RatingHard} {
		s, d, state, days := Review(2.0, 5.0, StateLearning, 0, rating, DefaultWeights)
		if state != StateLearning {
			t.Errorf("rating=%d: state = %d, want StateLearning(%d)", rating, state, StateLearning)
		}
		if days != 0 {
			t.Errorf("rating=%d: days = %d, want 0 (same-session re-review)", rating, days)
		}
		if s <= 0 {
			t.Errorf("rating=%d: stability = %f, want > 0", rating, s)
		}
		if d < 1 || d > 10 {
			t.Errorf("rating=%d: difficulty = %f, want in [1,10]", rating, d)
		}
	}
}

func TestLearningGraduateOnGoodEasy(t *testing.T) {
	// A Learning card rated Good or Easy graduates to Review with ≥1 day.
	for _, rating := range []int{RatingGood, RatingEasy} {
		s, d, state, days := Review(2.0, 5.0, StateLearning, 0, rating, DefaultWeights)
		if state != StateReview {
			t.Errorf("rating=%d: state = %d, want StateReview(%d)", rating, state, StateReview)
		}
		if days < 1 {
			t.Errorf("rating=%d: days = %d, want >= 1", rating, days)
		}
		if s <= 0 {
			t.Errorf("rating=%d: stability = %f, want > 0", rating, s)
		}
		if d < 1 || d > 10 {
			t.Errorf("rating=%d: difficulty = %f, want in [1,10]", rating, d)
		}
	}
}

func TestRelearningStayOnAgainHard(t *testing.T) {
	// A Relearning card rated Again or Hard stays in Relearning with 0 days.
	for _, rating := range []int{RatingAgain, RatingHard} {
		s, _, state, days := Review(5.0, 6.0, StateRelearning, 0.5, rating, DefaultWeights)
		if state != StateRelearning {
			t.Errorf("rating=%d: state = %d, want StateRelearning(%d)", rating, state, StateRelearning)
		}
		if days != 0 {
			t.Errorf("rating=%d: days = %d, want 0 (same-session re-review)", rating, days)
		}
		// Stability after failure should not grow.
		if s > 10 {
			t.Errorf("rating=%d: stability = %f, expected low after failure", rating, s)
		}
	}
}

func TestRelearningGraduateOnGoodEasy(t *testing.T) {
	// A Relearning card rated Good or Easy graduates back to Review with ≥1 day.
	for _, rating := range []int{RatingGood, RatingEasy} {
		s, _, state, days := Review(5.0, 6.0, StateRelearning, 0.5, rating, DefaultWeights)
		if state != StateReview {
			t.Errorf("rating=%d: state = %d, want StateReview(%d)", rating, state, StateReview)
		}
		if days < 1 {
			t.Errorf("rating=%d: days = %d, want >= 1", rating, days)
		}
		if s <= 0 {
			t.Errorf("rating=%d: stability = %f, want > 0", rating, s)
		}
	}
}

func TestReviewAgainReturnsZeroDays(t *testing.T) {
	// Regression: Review+Again must return 0 days for same-session re-review.
	s, d, state, days := Review(10.0, 5.0, StateReview, 10.0, RatingAgain, DefaultWeights)
	if state != StateRelearning {
		t.Errorf("state = %d, want StateRelearning(%d)", state, StateRelearning)
	}
	if days != 0 {
		t.Errorf("days = %d, want 0 (immediate re-review after lapse)", days)
	}
	// Stability should drop significantly after failure.
	if s > 10.0 {
		t.Errorf("stability after failure = %f, expected <= 10 (drop from prior 10)", s)
	}
	if d < 1 || d > 10 {
		t.Errorf("difficulty = %f, want in [1,10]", d)
	}
}

func TestElapsedDaysClamp(t *testing.T) {
	// Negative elapsedDays → clamped to 0 (e.g. clock skew or bad data).
	sNeg, _, stateNeg, _ := Review(10.0, 5.0, StateReview, -5.0, RatingGood, DefaultWeights)
	if stateNeg != StateReview {
		t.Errorf("negative elapsed: state = %d, want StateReview", stateNeg)
	}
	if sNeg <= 0 {
		t.Error("negative elapsed: stability should be > 0")
	}

	// Very large elapsedDays → capped at maxInterval.
	sHuge, _, stateHuge, _ := Review(10.0, 5.0, StateReview, 100000.0, RatingGood, DefaultWeights)
	if stateHuge != StateReview {
		t.Errorf("huge elapsed: state = %d, want StateReview", stateHuge)
	}
	if sHuge <= 0 {
		t.Error("huge elapsed: stability should be > 0")
	}
}

func TestConsecutiveFailuresDecayStability(t *testing.T) {
	// Multiple Again ratings should steadily decrease stability.
	s, d, state := NewCardState(DefaultWeights)
	// First review: Good to establish baseline.
	s, d, state, _ = Review(s, d, state, 0, RatingGood, DefaultWeights)
	baselineDays := int(math.Round(s))

	// Second review: Again → drops to Relearning.
	s, d, state, _ = Review(s, d, state, float64(baselineDays), RatingAgain, DefaultWeights)
	if state != StateRelearning {
		t.Fatalf("after first Again: state = %d, want Relearning", state)
	}
	afterFirstFail := s

	// Third review: Again (still in Relearning) → should drop further or stay low.
	s, _, state, _ = Review(s, d, state, 0, RatingAgain, DefaultWeights)
	if s > afterFirstFail*1.1 {
		t.Errorf("consecutive Again: stability %f > previous %f, expected decay", s, afterFirstFail)
	}
	if state != StateRelearning {
		t.Errorf("after second Again: state = %d, want Relearning", state)
	}
}

func TestNextReviewTime(t *testing.T) {
	now := time.Date(2026, 5, 19, 14, 30, 0, 0, time.UTC)

	// days=0 → immediately past (due right now).
	t0 := NextReviewTime(now, 0)
	if !t0.Before(now) {
		t.Errorf("NextReviewTime(now, 0) = %s, want before now", t0)
	}

	// days=1 → next day 00:00.
	t1 := NextReviewTime(now, 1)
	expected := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	if !t1.Equal(expected) {
		t.Errorf("NextReviewTime(now, 1) = %s, want %s", t1, expected)
	}

	// days=-1 (caller mistake) → treated as 0, immediately past.
	tNeg := NextReviewTime(now, -1)
	if !tNeg.Before(now) {
		t.Errorf("NextReviewTime(now, -1) = %s, want before now", tNeg)
	}
}

func TestDueZeroTime(t *testing.T) {
	// A zero time means the card has never been reviewed — should be due.
	if !Due(time.Time{}) {
		t.Error("zero time (never reviewed) should be due")
	}
}

func TestDaysLateFuture(t *testing.T) {
	// A card due in the future should have negative days late.
	future := time.Now().UTC().Add(48 * time.Hour)
	days := DaysLate(future)
	if days >= 0 {
		t.Errorf("DaysLate for future = %f, want negative", days)
	}
}

func TestRetrievabilityDecay(t *testing.T) {
	// After half the stability period, retrievability should be between 0.9 and 1.0.
	r := retrievability(5, 10)
	// After 5 days with 10-day stability: R = 0.9^(5/10) ≈ 0.949
	if r < 0.9 || r > 1.0 {
		t.Errorf("retrievability(5, 10) = %f, want in [0.9, 1.0]", r)
	}

	// After twice the stability, retrievability should be below 0.9.
	r = retrievability(20, 10)
	if r > 0.81 { // 0.9^2 = 0.81
		t.Errorf("retrievability(20, 10) = %f, want <= 0.81", r)
	}
}

func TestRatingConstants(t *testing.T) {
	// Verify rating constants are as expected (FSRS-5 standard).
	if RatingAgain != 1 || RatingHard != 2 || RatingGood != 3 || RatingEasy != 4 {
		t.Error("rating constants mismatch")
	}
	if StateNew != 0 || StateLearning != 1 || StateReview != 2 || StateRelearning != 3 {
		t.Error("state constants mismatch")
	}
}
