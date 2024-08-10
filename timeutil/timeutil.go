package timeutil

import (
	"fmt"
	"time"
)

const (
	Month        = time.Hour * 24 * 30
	Day          = time.Hour * 24
	RFC3339Milli = "YYYY-MM-DDTHH:MM:SS.sssZ"
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
