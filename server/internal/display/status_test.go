package display

import (
	"image/color"
	"testing"
	"time"
)

func TestArrivalStatusOf(t *testing.T) {
	scheduled := time.Date(2026, 3, 8, 8, 25, 0, 0, time.Local)

	tests := []struct {
		name     string
		expected time.Time
		realtime bool
		want     arrivalStatus
	}{
		{
			name:     "no realtime data",
			expected: scheduled,
			realtime: false,
			want:     arrivalUnknown,
		},
		{
			name:     "on time",
			expected: scheduled.Add(30 * time.Second),
			realtime: true,
			want:     arrivalOnTime,
		},
		{
			name:     "late",
			expected: scheduled.Add(3 * time.Minute),
			realtime: true,
			want:     arrivalLate,
		},
		{
			name:     "early",
			expected: scheduled.Add(-2 * time.Minute),
			realtime: true,
			want:     arrivalEarly,
		},
	}

	for _, test := range tests {
		if got := arrivalStatusOf(
			test.expected,
			scheduled,
			test.realtime,
		); got != test.want {
			t.Errorf("%s: arrivalStatusOf = %v, want %v", test.name, got, test.want)
		}
	}
}

func TestStatusColor(t *testing.T) {
	green := color.RGBA{R: 74, G: 222, B: 128, A: 255}
	red := color.RGBA{R: 239, G: 68, B: 68, A: 255}
	white := color.RGBA{R: 235, G: 240, B: 245, A: 255}

	tests := []struct {
		status arrivalStatus
		want   color.RGBA
	}{
		{arrivalOnTime, green},
		{arrivalEarly, green},
		{arrivalLate, red},
		{arrivalUnknown, white},
	}

	for _, test := range tests {
		if got := test.status.statusColor(); got != test.want {
			t.Errorf("statusColor(%v) = %v, want %v", test.status, got, test.want)
		}
	}
}
