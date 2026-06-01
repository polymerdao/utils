package utils

import (
	"testing"
	"time"
)

func newTimerFixture(t *testing.T) (*TestTimer, time.Time) {
	t.Helper()
	now, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("parse base time: %v", err)
	}
	return NewTestTimer(now.Format(time.RFC3339)), now
}

func TestTimerNow(t *testing.T) {
	timer, now := newTimerFixture(t)

	if got := timer.Now(); !got.Equal(now) {
		t.Fatalf("Now() = %v, want %v", got, now)
	}

	timer.Advance(2 * time.Hour)
	if got, want := timer.Now(), now.Add(2*time.Hour); !got.Equal(want) {
		t.Fatalf("after Advance: Now() = %v, want %v", got, want)
	}

	timer.SetTime("2024-06-15T12:30:00Z")
	if got := timer.Now().Format(time.RFC3339); got != "2024-06-15T12:30:00Z" {
		t.Fatalf("after SetTime: Now() = %s, want 2024-06-15T12:30:00Z", got)
	}
}

func TestTimerInvalidSetTime(t *testing.T) {
	timer, _ := newTimerFixture(t)

	for _, in := range []string{"invalid-time-format", "2024 06 15"} {
		t.Run(in, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("SetTime(%q) did not panic", in)
				}
			}()
			timer.SetTime(in)
		})
	}
}

func TestTimerAdd(t *testing.T) {
	timer, now := newTimerFixture(t)

	d := 90 * time.Minute
	if got, want := timer.Add(d), now.Add(d); !got.Equal(want) {
		t.Fatalf("Add(%v) = %v, want %v", d, got, want)
	}

	// Add() does not modify the internal time of the timer
	if got := timer.Now(); !got.Equal(now) {
		t.Fatalf("Now() after Add = %v, want %v", got, now)
	}
}
