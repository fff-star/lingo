package store

import (
	"database/sql"
	"time"
)

type sqliteReviewLog struct {
	db *sql.DB
}

func NewReviewLog(db *sql.DB) ReviewLog {
	return &sqliteReviewLog{db: db}
}

func (l *sqliteReviewLog) Record(date string) error {
	_, err := l.db.Exec(`INSERT INTO review_log (date, count) VALUES (?, 1)
		ON CONFLICT(date) DO UPDATE SET count = count + 1`, date)
	return err
}

func (l *sqliteReviewLog) TodayCount() (int, error) {
	today := time.Now().UTC().Format("2006-01-02")
	var count int
	err := l.db.QueryRow("SELECT COALESCE(count, 0) FROM review_log WHERE date = ?", today).Scan(&count)
	return count, err
}

func (l *sqliteReviewLog) Streak() (current, longest int, err error) {
	rows, err := l.db.Query("SELECT date FROM review_log ORDER BY date DESC")
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return 0, 0, err
		}
		dates = append(dates, d)
	}

	if len(dates) == 0 {
		return 0, 0, nil
	}

	today := time.Now().UTC().Format("2006-01-02")
	currentStreak := 0
	longestStreak := 0

	// Walk backwards from today to find current streak.
	checkDate := time.Now().UTC()
	for {
		dateStr := checkDate.Format("2006-01-02")
		found := false
		for _, d := range dates {
			if d == dateStr {
				found = true
				break
			}
		}
		if found {
			currentStreak++
		} else if dateStr != today {
			break
		}
		checkDate = checkDate.AddDate(0, 0, -1)
	}

	// Find longest streak by scanning all dates.
	allDates := make(map[string]bool)
	for _, d := range dates {
		allDates[d] = true
	}
	for _, d := range dates {
		t, _ := time.Parse("2006-01-02", d)
		run := 1
		for {
			t = t.AddDate(0, 0, -1)
			if allDates[t.Format("2006-01-02")] {
				run++
			} else {
				break
			}
		}
		if run > longestStreak {
			longestStreak = run
		}
	}

	return currentStreak, longestStreak, nil
}

func (l *sqliteReviewLog) Stats(newCounts map[string]int) *ReviewStats {
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
		dateKey := now.AddDate(0, 0, -(14 - i)).Format("2006-01-02")
		stats.New[i] = newCounts[dateKey]

		var count int
		l.db.QueryRow("SELECT count FROM review_log WHERE date = ?", dateKey).Scan(&count)
		stats.Reviews[i] = count
	}

	return stats
}
