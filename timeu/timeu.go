package timeu

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jptrs93/goutil/contextu"
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

type Backoff struct {
	CurrentDuration time.Duration
	MaxDuration     time.Duration
	F               func(time.Duration) time.Duration
}

func (b *Backoff) Wait(ctx context.Context) {
	b.CurrentDuration = b.F(b.CurrentDuration)
	if b.MaxDuration > 0 && b.CurrentDuration > b.MaxDuration {
		b.CurrentDuration = b.MaxDuration
	}
	contextu.Sleep(ctx, b.CurrentDuration)
}

func (b *Backoff) Reset() {
	b.CurrentDuration = 0
}

func NewExpBackoff(maxDuration time.Duration) *Backoff {
	return &Backoff{
		CurrentDuration: 0,
		MaxDuration:     maxDuration,
		F: func(i time.Duration) time.Duration {
			return max(i, time.Second) * 2
		},
	}
}

func NewLinearBackoff(increment, maxDuration time.Duration) *Backoff {
	return &Backoff{
		CurrentDuration: 0,
		MaxDuration:     maxDuration,
		F: func(i time.Duration) time.Duration {
			return i + increment
		},
	}
}
