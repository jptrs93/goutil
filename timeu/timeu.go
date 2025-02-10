package timeu

import (
	"fmt"
	"time"
)

const (
	Month        = time.Hour * 24 * 30
	Day          = time.Hour * 24
	RFC3339Milli = "2006-01-02T15:04:05.000Z"
)

func MustParse(val string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", val)
	if err != nil {
		panic(fmt.Sprintf("failed to parse time '%v': %v", val, err))
	}
	t = t.UTC()
	return t
}

func MinTime(v ...time.Time) time.Time {
	var winner time.Time
	if len(v) == 0 {
		return winner
	}
	winner = v[0]
	for _, t := range v[1:] {
		if t.Before(winner) {
			winner = t
		}
	}
	return winner
}

func MaxTime(v ...time.Time) time.Time {
	var winner time.Time
	if len(v) == 0 {
		return winner
	}
	winner = v[0]
	for _, t := range v[1:] {
		if t.After(winner) {
			winner = t
		}
	}
	return winner
}

func IsoWeekStartEnd(year int, week int) (time.Time, time.Time) {
	// January 4 is always in ISO week 1.
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	offset := (int(jan4.Weekday()) + 6) % 7
	startOfWeek1 := jan4.AddDate(0, 0, -offset)

	weekStart := startOfWeek1.AddDate(0, 0, (week-1)*7)

	calculatedYear, calculatedWeek := weekStart.ISOWeek()
	if calculatedYear != year || calculatedWeek != week || weekStart.Weekday() != time.Monday {
		panic("invalid year/week combination")
	}

	weekEnd := weekStart.AddDate(0, 0, 7)
	return weekStart, weekEnd
}

func TruncateIsoWeek(t time.Time) time.Time {
	y, w := t.ISOWeek()
	s, _ := IsoWeekStartEnd(y, w)
	return s
}
