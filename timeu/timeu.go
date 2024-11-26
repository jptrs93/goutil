package timeu

import (
	"fmt"
	"math/rand"
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
	if calculatedYear != year || calculatedWeek != week || weekStart.Weekday() != time.Monday {
		panic("invalid year/week combination")
	}

	weekEnd := weekStart.AddDate(0, 0, 6).Add(time.Hour * 24)
	return weekStart, weekEnd
}

func TruncateIsoWeek(t time.Time) time.Time {
	y, w := t.ISOWeek()
	s, _ := IsoWeekStartEnd(y, w)
	return s
}

type FixedImmediateTimer struct {
	NextTriggerTime time.Time
	Interval        time.Duration
	Offset          time.Duration
	Iterations      int
}

func NewTimerWithJitter(interval time.Duration) *FixedImmediateTimer {
	return NewFixedImmediateTimer(jitter(), interval)
}

func NewFixedImmediateTimer(offset, interval time.Duration) *FixedImmediateTimer {
	// the first tick is at epoch 0 + offset then every interval after that
	tNow := time.Now().UTC()
	t0 := time.Time{}.UTC().Add(offset)
	// note we have to use milli's as max duration is 290 years
	diffMilli := tNow.UnixMilli() - t0.UnixMilli()
	n := diffMilli / interval.Milliseconds()
	nextTriggerMilli := t0.UnixMilli() + (n+1)*interval.Milliseconds()
	nextTriggerTime := time.Unix(nextTriggerMilli/1000, (nextTriggerMilli%1000)*int64(time.Millisecond)).UTC()
	return &FixedImmediateTimer{
		NextTriggerTime: nextTriggerTime,
		Interval:        interval,
		Offset:          offset,
	}
}

func (t *FixedImmediateTimer) Wait() {
	t.Iterations += 1
	pauseDuration := t.NextTriggerTime.Sub(time.Now().UTC())
	// we want the timer to trigger immediately upon starting
	// so if the first pause is > 10s then just wait a random [0,5]s and return
	if t.Iterations == 1 && pauseDuration > time.Second*10 {
		time.Sleep(time.Duration(rand.Intn(5000)) * time.Millisecond)
		return
	}
	if pauseDuration > 0 {
		time.Sleep(pauseDuration)
	}
	t.NextTriggerTime = time.Now().Add(t.Interval)
}

func jitter() time.Duration {
	maxDuration := time.Hour
	randomDuration := time.Duration(rand.Int63n(int64(maxDuration)))
	return randomDuration
}
