package store

import (
	"sort"
	"sync"
	"time"
)

// jsonReviewLog records daily review counts.
type jsonReviewLog struct {
	mu       sync.Mutex
	filePath string
	data     map[string]int
}

// ReviewStats holds per-day statistics for chart display.
type ReviewStats struct {
	Labels  []string `json:"labels"`
	New     []int    `json:"new"`
	Reviews []int    `json:"reviews"`
}

func NewJSONReviewLog(filePath string) *jsonReviewLog {
	return &jsonReviewLog{filePath: filePath}
}

func (l *jsonReviewLog) TodayCount() (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	counts, _ := l.loadUnsafe()
	today := time.Now().UTC().Format("2006-01-02")
	return counts[today], nil
}

func (l *jsonReviewLog) Streak() (current, longest int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	counts, _ := l.loadUnsafe()

	var dates []string
	for d := range counts {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	if len(dates) == 0 {
		return 0, 0, nil
	}

	today := time.Now().UTC().Format("2006-01-02")
	currentStreak := 0
	checkDate := time.Now().UTC()
	for {
		dateStr := checkDate.Format("2006-01-02")
		if counts[dateStr] > 0 {
			currentStreak++
		} else if dateStr != today {
			break
		}
		checkDate = checkDate.AddDate(0, 0, -1)
	}

	longestStreak := 0
	currentRun := 0
	for i, d := range dates {
		if i == 0 {
			currentRun = 1
		} else {
			prev, _ := time.Parse("2006-01-02", dates[i-1])
			curr, _ := time.Parse("2006-01-02", d)
			if curr.Sub(prev).Hours() <= 24 {
				currentRun++
			} else {
				currentRun = 1
			}
		}
		if currentRun > longestStreak {
			longestStreak = currentRun
		}
	}

	return currentStreak, longestStreak, nil
}

// Record logs a review event for the given date.
func (l *jsonReviewLog) Record(date string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	counts, err := l.loadUnsafe()
	if err != nil {
		return err
	}
	counts[date] = counts[date] + 1
	if err := writeJSON(l.filePath, counts); err != nil {
		return err
	}
	l.data = counts
	return nil
}

// Stats returns the last 30 days of review counts plus new-item counts.
func (l *jsonReviewLog) Stats(newCounts map[string]int) *ReviewStats {
	l.mu.Lock()
	defer l.mu.Unlock()

	counts, _ := l.loadUnsafe()

	// Collect last 15 days.
	now := time.Now().UTC()
	var labels []string
	for i := 14; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		labels = append(labels, d.Format("01-02"))
	}

	stats := &ReviewStats{
		Labels:  labels,
		New:     make([]int, 15),
		Reviews: make([]int, 15),
	}
	for i := range labels {
		// Build the full date key matching loaded dates.
		dateKey := now.AddDate(0, 0, -(14-i)).Format("2006-01-02")
		stats.Reviews[i] = counts[dateKey]
		stats.New[i] = newCounts[dateKey]
	}

	return stats
}

func (l *jsonReviewLog) loadUnsafe() (map[string]int, error) {
	if l.data != nil {
		out := make(map[string]int, len(l.data))
		for k, v := range l.data {
			out[k] = v
		}
		return out, nil
	}
	var counts map[string]int
	if err := readJSON(l.filePath, &counts); err != nil {
		return make(map[string]int), nil
	}
	if counts == nil {
		counts = make(map[string]int)
	}
	l.data = counts
	return l.data, nil
}

// NewCountsByDate returns per-day counts of items created on each day.
// items should have CreatedAt time.Time accessible. We use a simple callback.
func NewCountsByDate(createdDates []time.Time) map[string]int {
	counts := make(map[string]int)
	for _, d := range createdDates {
		key := d.UTC().Format("2006-01-02")
		counts[key]++
	}
	return counts
}

// MergeCounts merges multiple count maps.
func MergeCounts(maps ...map[string]int) map[string]int {
	result := make(map[string]int)
	for _, m := range maps {
		for k, v := range m {
			result[k] += v
		}
	}
	return result
}

// SortedKeys returns sorted keys of a map.
func SortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
