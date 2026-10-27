package NetBirdSDK

import (
	"testing"
	"time"
)

func TestSessionExpiresAtUnix(t *testing.T) {
	deadline := time.Date(2026, 10, 27, 18, 30, 0, 0, time.UTC)
	tests := []struct {
		name     string
		deadline time.Time
		want     int64
	}{
		{name: "set", deadline: deadline, want: deadline.Unix()},
		{name: "set in another zone", deadline: deadline.In(time.FixedZone("UTC+4", 4*3600)), want: deadline.Unix()},
		{name: "past is returned as-is", deadline: time.Unix(1_000, 0), want: 1_000},
		{name: "zero", deadline: time.Time{}, want: 0},
		{name: "epoch", deadline: time.Unix(0, 0), want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionExpiresAtUnix(tt.deadline); got != tt.want {
				t.Errorf("sessionExpiresAtUnix(%v) = %d, want %d", tt.deadline, got, tt.want)
			}
		})
	}
}
