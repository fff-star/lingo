package store

import (
	"sort"
	"sync"
	"time"
)

// ReviewLog records daily review counts.
type ReviewLog struct {
	mu       sync.Mutex
	filePath string
}

// ReviewStats holds per-day statistics for chart display.
type ReviewStats struct {
	Labels  []string `json:"labels"`
	New     []int    `json:"new"`
	Reviews []int    `json:"reviews"`
}

func NewReviewLog(filePath string) *ReviewLog {
	return &ReviewLog{filePath: filePath}
}

// Record logs a review event for the given date.
func (l *ReviewLog) Record(date string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	counts, err := l.loadUnsafe()
	if err != nil {
		return err
	}
	counts[date] = counts[date] + 1
	return writeJSON(l.filePath, counts)
}

// Stats returns the last 30 days of review counts plus new-item counts.
func (l *ReviewLog) Stats(newCounts map[string]int) *ReviewStats {
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

func (l *ReviewLog) loadUnsafe() (map[string]int, error) {
	var counts map[string]int
	if err := readJSON(l.filePath, &counts); err != nil {
		return make(map[string]int), nil
	}
	if counts == nil {
		counts = make(map[string]int)
	}
	return counts, nil
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
