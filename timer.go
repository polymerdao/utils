package utils

import "time"

// Timer defines an interface for getting the current time
// helpful for testing
type Timer interface {
	Now() time.Time
}

// sysTimer is a Timer implementation that uses the system time
type sysTimer struct{}

var _ Timer = (*sysTimer)(nil)

// SystemTimer returns a Timer that uses the system time
func SystemTimer() Timer {
	return sysTimer{}
}

// Now returns the current system time
func (sysTimer) Now() time.Time {
	return time.Now()
}

// TestTimer is a Timer implementation for testing purposes
type TestTimer struct {
	now time.Time
}

var _ Timer = (*TestTimer)(nil)

// NewTestTimer creates a new TestTimer initialized to the specified RFC3339 time string
func NewTestTimer(currentRCF3339Time string) *TestTimer {
	t := &TestTimer{}
	t.SetTime(currentRCF3339Time)
	return t
}

// Now returns the current time set in the TestTimer
func (t *TestTimer) Now() time.Time {
	return t.now
}

// Advance advances the current time by the specified duration
func (t *TestTimer) Advance(d time.Duration) {
	t.now = t.now.Add(d)
}

// SetTime sets the current time to the specified RFC3339 time string
func (t *TestTimer) SetTime(rfc3339Time string) {
	now, err := time.Parse(time.RFC3339, rfc3339Time)
	if err != nil {
		panic("invalid time format for TestTimer: " + err.Error())
	}
	t.now = now
}

// Add adds the specified duration to the current time and returns the new time
func (t *TestTimer) Add(d time.Duration) time.Time {
	return t.now.Add(d)
}
