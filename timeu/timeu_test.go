package timeu

import (
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
