package timeu

import (
	"context"
	"testing"
	"time"
)

func TestNextOffsetTickerTick(t *testing.T) {
	period := time.Hour
	around := time.Date(2026, time.May, 7, 0, 15, 0, 0, time.UTC)

	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "next tick later in same period",
			now:  time.Date(2026, time.May, 7, 10, 3, 0, 0, time.UTC),
			want: time.Date(2026, time.May, 7, 10, 15, 0, 0, time.UTC),
		},
		{
			name: "exact tick advances to future tick",
			now:  time.Date(2026, time.May, 7, 10, 15, 0, 0, time.UTC),
			want: time.Date(2026, time.May, 7, 11, 15, 0, 0, time.UTC),
		},
		{
			name: "anchor itself is future tick",
			now:  time.Date(2026, time.May, 6, 23, 55, 0, 0, time.UTC),
			want: around,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextOffsetTickerTick(tt.now, period, around)
			if !got.Equal(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDailyScheduleNextAfter(t *testing.T) {
	schedule := DailySchedule{Hour: 4, Minute: 30, Second: 15, Location: time.UTC}

	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "later same day",
			now:  time.Date(2026, time.May, 7, 3, 0, 0, 0, time.UTC),
			want: time.Date(2026, time.May, 7, 4, 30, 15, 0, time.UTC),
		},
		{
			name: "exact time advances to tomorrow",
			now:  time.Date(2026, time.May, 7, 4, 30, 15, 0, time.UTC),
			want: time.Date(2026, time.May, 8, 4, 30, 15, 0, time.UTC),
		},
		{
			name: "later tomorrow",
			now:  time.Date(2026, time.May, 7, 5, 0, 0, 0, time.UTC),
			want: time.Date(2026, time.May, 8, 4, 30, 15, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schedule.NextAfter(tt.now)
			if !got.Equal(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWeeklyScheduleNextAfter(t *testing.T) {
	schedule := WeeklySchedule{Weekday: time.Friday, Hour: 4, Location: time.UTC}

	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "later same week",
			now:  time.Date(2026, time.May, 7, 3, 0, 0, 0, time.UTC), // Thursday
			want: time.Date(2026, time.May, 8, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "later same day",
			now:  time.Date(2026, time.May, 8, 3, 0, 0, 0, time.UTC),
			want: time.Date(2026, time.May, 8, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "exact time advances to next week",
			now:  time.Date(2026, time.May, 8, 4, 0, 0, 0, time.UTC),
			want: time.Date(2026, time.May, 15, 4, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schedule.NextAfter(tt.now)
			if !got.Equal(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNextScheduleTickerTickAppliesStableJitter(t *testing.T) {
	schedule := DailySchedule{Hour: 4, Location: time.UTC}
	jitter := 17 * time.Minute
	now := time.Date(2026, time.May, 7, 3, 0, 0, 0, time.UTC)

	got := nextScheduleTickerTick(now, schedule, jitter)
	want := time.Date(2026, time.May, 7, 4, 17, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	got = nextScheduleTickerTick(want, schedule, jitter)
	want = time.Date(2026, time.May, 8, 4, 17, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBackoffWaitWithContextResetsAfterResetDuration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fCalled := false
	b := &Backoff{
		CurrentDuration: 5 * time.Second,
		ResetDuration:   time.Hour,
		lastWait:        time.Now().Add(-2 * time.Hour),
		F: func(i time.Duration) time.Duration {
			fCalled = true
			return time.Second
		},
	}

	b.WaitWithContext(ctx)
	if fCalled {
		t.Fatal("expected reset without backoff function call")
	}
	if b.CurrentDuration != 0 {
		t.Fatalf("current duration = %v, want 0", b.CurrentDuration)
	}
	if b.lastWait.IsZero() {
		t.Fatal("expected reset wait to update last wait time")
	}

	b.WaitWithContext(ctx)
	if !fCalled {
		t.Fatal("expected next wait within reset duration to call backoff function")
	}
	if b.CurrentDuration != time.Second {
		t.Fatalf("current duration = %v, want %v", b.CurrentDuration, time.Second)
	}
}
