package timeutil

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

	startDate := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	_, isoWeek := startDate.ISOWeek()

	if isoWeek != 1 {
		startDate = startDate.AddDate(0, 0, (8-int(startDate.Weekday()))%7) // Move to the next Monday
		_, isoWeek = startDate.ISOWeek()
	}

	weekDifference := week - isoWeek
	weekStart := startDate.AddDate(0, 0, weekDifference*7)

	calculatedYear, calculatedWeek := weekStart.ISOWeek()
	if calculatedYear != year || calculatedWeek != week {
		panic("invalid year/week combination")
	}

	weekEnd := weekStart.AddDate(0, 0, 6).Add(time.Hour * 24)
	return weekStart, weekEnd
}
