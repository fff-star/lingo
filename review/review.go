package review

import (
	"math"
	"time"
)

// FSRS-5 (Free Spaced Repetition Scheduler) implementation.
// Based on the algorithm by Jarrett Ye (L.M.Sherlock), now the default in Anki.
//
// Core idea: memory is modeled with three components:
//   - Stability (S): how many days until retrievability drops to 90%
//   - Difficulty (D): inherent difficulty of the material (1–10)
//   - Retrievability (R): probability of recall at a given moment

// Rating constants.
const (
	RatingAgain = 1
	RatingHard  = 2
	RatingGood  = 3
	RatingEasy  = 4
)

// State constants.
const (
	StateNew        = 0
	StateLearning   = 1
	StateReview     = 2
	StateRelearning = 3
)

// DefaultWeights are the FSRS-5 default parameters.
// w[0..3]:  initial stability after Again/Hard/Good/Easy
// w[4]:     initial difficulty
// w[5]:     difficulty delta for Again
// w[6]:     difficulty delta for Hard
// w[7]:     difficulty damping (mean reversion)
// w[8]:     stability increase factor
// w[9]:     stability exponent (negative)
// w[10]:    retrievability impact on stability increase
// w[11]:    stability after failure factor
// w[12]:    stability exponent after failure
// w[13]:    retrievability impact on failure stability
// w[14]:    desired retention
// w[15]:    easy bonus multiplier
// w[16]:    hard penalty multiplier
var DefaultWeights = [17]float64{
	0.4026,  // w0
	1.1839,  // w1
	3.136,   // w2
	15.0105, // w3
	5.0,     // w4
	0.2,     // w5
	0.8,     // w6
	1.2,     // w7
	1.5,     // w8
	0.1,     // w9
	0.8,     // w10
	1.0,     // w11
	0.5,     // w12
	0.6,     // w13
	0.9,     // w14
	1.3,     // w15
	1.2,     // w16
}

const maxInterval = 36500 // ~100 years

// NewCardState returns the initial FSRS state for a brand-new card.
// Fields to set on the model:
//
//	Stability:  0
//	Difficulty: w[4]  (default 5.0)
//	State:      StateNew (0)
func NewCardState(w [17]float64) (stability, difficulty float64, state int) {
	return 0, w[4], StateNew
}

// Review processes one review and returns the updated FSRS state plus
// how many days until the next review is due.
//
// Parameters:
//
//	stability   — current stability (0 for new cards)
//	difficulty  — current difficulty (w[4] for new cards)
//	state       — current state (StateNew for new cards)
//	elapsedDays — days since the last review (0 for new cards)
//	rating      — user rating: Again=1, Hard=2, Good=3, Easy=4
//	w           — FSRS weights (pass DefaultWeights)
func Review(stability, difficulty float64, state int, elapsedDays float64, rating int, w [17]float64) (newStability, newDifficulty float64, newState int, scheduledDays int) {
	// ── First review of a new/learning card: use initial stability ──
	if state == StateNew || state == StateLearning {
		newS := initialStabilityForRating(rating, w)
		newD := updateDifficulty(difficulty, rating, w)
		newD = clamp(newD, 1, 10)
		days := int(math.Round(newS))
		return newS, newD, StateReview, max(days, 1)
	}

	// ── Retrievability: probability of recall right now ──
	R := retrievability(elapsedDays, stability)

	// ── Update difficulty ──
	newD := updateDifficulty(difficulty, rating, w)
	newD = clamp(newD, 1, 10)

	// ── Update stability ──
	var newS float64

	if rating == RatingAgain {
		newS = stabilityAfterFailure(stability, R, w)
		newState = StateRelearning
	} else {
		// Hard/Good/Easy share the success path, then apply
		// hard-penalty or easy-bonus multipliers.
		S := stabilityAfterSuccess(stability, newD, R, w)

		if rating == RatingHard {
			S *= w[16] // hard penalty
		}
		if rating == RatingEasy {
			S *= w[15] // easy bonus
		}
		newS = S
		newState = StateReview
	}

	// ── Scheduled interval from stability ──
	days := int(math.Round(newS))
	if days < 1 {
		days = 1
	}
	if days > maxInterval {
		days = maxInterval
	}

	return newS, newD, newState, days
}

// NextInterval computes the ideal interval from a stability value.
func NextInterval(stability float64, desiredRetention float64) int {
	if stability <= 0 {
		return 1
	}
	factor := math.Log(desiredRetention) / math.Log(0.9)
	days := int(math.Round(stability * factor))
	if days < 1 {
		days = 1
	}
	if days > maxInterval {
		days = maxInterval
	}
	return days
}

// ─────── internal helpers ───────

func initialStabilityForRating(rating int, w [17]float64) float64 {
	switch rating {
	case RatingAgain:
		return w[0]
	case RatingHard:
		return w[1]
	case RatingGood:
		return w[2]
	case RatingEasy:
		return w[3]
	default:
		return w[2]
	}
}

// updateDifficulty applies the mean-reversion update from FSRS.
func updateDifficulty(D float64, rating int, w [17]float64) float64 {
	// ΔD = -w[5] * (G - 3)  for G ≥ 3; else ΔD = -w[6] * (G - 3)
	var delta float64
	if rating >= RatingGood {
		delta = -w[5] * float64(rating-3) // Good=0, Easy=-w[5]
	} else {
		delta = -w[6] * float64(rating-3) // Again=+2*w[6], Hard=+w[6]
	}
	// Mean reversion: D' = D + w[7] * delta * (10 - D) / 9  (pull toward 10 for positive delta)
	//                D' = D + w[7] * delta * (D - 1) / 9    (pull toward 1 for negative delta)
	if delta >= 0 {
		return D + w[7]*delta*(10-D)/9
	}
	return D + w[7]*delta*(D-1)/9
}

func stabilityAfterSuccess(S, D, R float64, w [17]float64) float64 {
	hardPart := math.Exp(w[10] * (1 - R))
	// (11 - D) represents: easier cards (low D) get bigger stability gains
	SInc := 1 + math.Exp(w[8])*(11-D)*math.Pow(S, -w[9])*(hardPart-1)
	return S * max(SInc, 1)
}

func stabilityAfterFailure(S, R float64, w [17]float64) float64 {
	return w[11] * math.Pow(S, w[12]) * math.Exp(w[13]*(1-R))
}

func retrievability(elapsedDays, stability float64) float64 {
	if stability <= 0 {
		return 1 // new card, no memory yet
	}
	return math.Exp(math.Log(0.9) * elapsedDays / stability)
}

func clamp(v, lo, hi float64) float64 {
	return max(lo, min(hi, v))
}

// Due returns true when a card is due for review.
// A zero nextReviewAt means the card has never been reviewed — treat it as due now.
func Due(nextReviewAt time.Time) bool {
	return nextReviewAt.IsZero() || time.Now().UTC().After(nextReviewAt)
}

// DaysLate returns how many days overdue a card is (negative = not due yet).
func DaysLate(nextReviewAt time.Time) float64 {
	return time.Now().UTC().Sub(nextReviewAt).Hours() / 24
}

// NextDayStart returns the start of day (00:00:00 in t's timezone) offset by days.
// 00:00–01:00 is treated as the previous day, so a review at 00:30 doesn't
// push the next review an extra day forward.
// The result is guaranteed to be after t, so a 1-day interval never lands
// on a time already past.
func NextDayStart(t time.Time, days int) time.Time {
	orig := t
	if t.Hour() == 0 {
		t = t.Add(-1 * time.Hour) // push back to previous day
	}
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	result := start.AddDate(0, 0, days)
	if !result.After(orig) {
		result = result.AddDate(0, 0, 1)
	}
	return result
}
