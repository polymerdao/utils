package utils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"testing"
	"time"
)

func newRetryFixture() (context.Context, *slog.Logger) {
	return context.Background(), slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func TestRetryNoError(t *testing.T) {
	ctx, log := newRetryFixture()
	retrier := NewConstantRetrier[int](ctx, log).Build()
	ret, err := retrier.Retry(func(context.Context) (int, error) {
		return 1, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ret != 1 {
		t.Fatalf("ret = %d, want 1", ret)
	}
}

func TestRetry(t *testing.T) {
	ctx, log := newRetryFixture()
	var retries int
	retrier := NewConstantRetrier[int](ctx, log).WithLogging(slog.LevelInfo).WithInterval(time.Millisecond).Build()
	ret, err := retrier.Retry(func(context.Context) (int, error) {
		if retries == 5 {
			return 1, nil
		}
		retries++
		return 0, fmt.Errorf("foo")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ret != 1 {
		t.Fatalf("ret = %d, want 1", ret)
	}
	if retries != 5 {
		t.Fatalf("retries = %d, want 5", retries)
	}
}

func TestRetryWithContext(t *testing.T) {
	ctx, log := newRetryFixture()
	var retries int
	retrier := NewConstantRetrier[int](ctx, log).
		WithLogging(slog.LevelInfo).
		WithContextTimeout(time.Second).
		WithInterval(time.Millisecond).
		Build()
	ret, err := retrier.Retry(func(context.Context) (int, error) {
		if retries == 5 {
			return 1, nil
		}
		retries++
		return 0, fmt.Errorf("foo")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ret != 1 {
		t.Fatalf("ret = %d, want 1", ret)
	}
	if retries != 5 {
		t.Fatalf("retries = %d, want 5", retries)
	}
}

func TestRetryWithContextTimeout(t *testing.T) {
	ctx, log := newRetryFixture()
	var retries int
	retrier := NewConstantRetrier[int](ctx, log).
		WithLogging(slog.LevelInfo).
		WithContextTimeout(5 * time.Millisecond).
		WithInterval(10 * time.Millisecond).
		Build()
	ret, err := retrier.Retry(func(context.Context) (int, error) {
		if retries == 5 {
			return 1, nil
		}
		retries++
		return 0, fmt.Errorf("foo")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ret != 0 {
		t.Fatalf("ret = %d, want 0", ret)
	}
}

func TestExponentialRetrier(t *testing.T) {
	ctx, log := newRetryFixture()
	var retries int
	retrier := NewExponentialRetrier[int](ctx, log).
		WithLogging(slog.LevelInfo).
		WithInterval(10 * time.Millisecond).
		WithMaxRetries(5).
		Build()
	ret, err := retrier.Retry(func(context.Context) (int, error) {
		if retries == 3 {
			return 42, nil
		}
		retries++
		return 0, fmt.Errorf("retry %d", retries)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ret != 42 {
		t.Fatalf("ret = %d, want 42", ret)
	}
	if retries != 3 {
		t.Fatalf("retries = %d, want 3", retries)
	}
}

func TestExponentialRetrierWithMultiplier(t *testing.T) {
	ctx, log := newRetryFixture()
	var retries int
	var intervals []time.Duration
	start := time.Now()
	var lastCheck time.Time

	retrier := NewExponentialRetrier[int](ctx, log).
		WithMultiplier(2.0).
		WithLogging(slog.LevelInfo).
		WithInterval(100 * time.Millisecond).
		WithMaxRetries(4).
		Build()

	ret, err := retrier.Retry(func(context.Context) (int, error) {
		now := time.Now()
		if !lastCheck.IsZero() {
			intervals = append(intervals, now.Sub(lastCheck))
		} else {
			lastCheck = start
		}
		lastCheck = now

		if retries == 3 {
			return 100, nil
		}
		retries++
		return 0, fmt.Errorf("retry %d", retries)
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ret != 100 {
		t.Fatalf("ret = %d, want 100", ret)
	}
	if retries != 3 {
		t.Fatalf("retries = %d, want 3", retries)
	}
	if len(intervals) != 3 {
		t.Fatalf("len(intervals) = %d, want 3", len(intervals))
	}

	tolerance := 50 * time.Millisecond
	assertInDelta(t, 100*time.Millisecond, intervals[0], tolerance, "first interval should be ~100ms")
	assertInDelta(t, 200*time.Millisecond, intervals[1], tolerance, "second interval should be ~200ms (100ms * 2)")
	assertInDelta(t, 400*time.Millisecond, intervals[2], tolerance, "third interval should be ~400ms (200ms * 2)")
}

func TestExponentialRetrierWithMaxInterval(t *testing.T) {
	ctx, log := newRetryFixture()
	var retries int
	var intervals []time.Duration
	start := time.Now()
	var lastCheck time.Time

	retrier := NewExponentialRetrier[int](ctx, log).
		WithMultiplier(3.0).
		WithMaxInterval(250 * time.Millisecond).
		WithLogging(slog.LevelInfo).
		WithInterval(100 * time.Millisecond).
		WithMaxRetries(5).
		Build()

	ret, err := retrier.Retry(func(context.Context) (int, error) {
		now := time.Now()
		if !lastCheck.IsZero() {
			intervals = append(intervals, now.Sub(lastCheck))
		} else {
			lastCheck = start
		}
		lastCheck = now

		if retries == 4 {
			return 200, nil
		}
		retries++
		return 0, fmt.Errorf("retry %d", retries)
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ret != 200 {
		t.Fatalf("ret = %d, want 200", ret)
	}
	if retries != 4 {
		t.Fatalf("retries = %d, want 4", retries)
	}
	if len(intervals) != 4 {
		t.Fatalf("len(intervals) = %d, want 4", len(intervals))
	}

	tolerance := 50 * time.Millisecond
	assertInDelta(t, 100*time.Millisecond, intervals[0], tolerance, "first interval should be ~100ms")
	assertInDelta(t, 250*time.Millisecond, intervals[1], tolerance, "second interval should be capped at ~250ms")
	assertInDelta(t, 250*time.Millisecond, intervals[2], tolerance, "third interval should be capped at ~250ms")
	assertInDelta(t, 250*time.Millisecond, intervals[3], tolerance, "fourth interval should be capped at ~250ms")
}

func TestCancelableRetrierStopsWhileWaiting(t *testing.T) {
	ctx, log := newRetryFixture()
	stop, cancel := context.WithCancel(context.Background())
	defer cancel()

	var attempts int
	retrier := NewCancelableConstantRetrier[int](stop, ctx, log).
		WithLogging(slog.LevelInfo).
		WithInterval(time.Hour). // the test hangs if the wait is not interrupted
		WithMaxRetries(0).
		Build()

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	ret, err := retrier.Retry(func(context.Context) (int, error) {
		attempts++
		return 7, fmt.Errorf("boom")
	})

	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("Retry took %v, want it to end as soon as the stop context is cancelled", elapsed)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want it to wrap context.Canceled", err)
	}
	if ret != 7 {
		t.Fatalf("ret = %d, want the last attempt's value 7", ret)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestCancelableRetrierAlwaysAttemptsOnce(t *testing.T) {
	ctx, log := newRetryFixture()
	stop, cancel := context.WithCancel(context.Background())
	cancel()

	var attempts int
	retrier := NewCancelableConstantRetrier[int](stop, ctx, log).
		WithInitialWait(time.Hour). // the initial wait is interruptible too
		WithInterval(time.Hour).
		Build()

	start := time.Now()
	ret, err := retrier.Retry(func(context.Context) (int, error) {
		attempts++
		return 3, fmt.Errorf("boom")
	})

	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("Retry took %v, want the initial wait to be interrupted", elapsed)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want the operation to always run once", attempts)
	}
	if ret != 3 {
		t.Fatalf("ret = %d, want the last attempt's value 3", ret)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want it to wrap context.Canceled", err)
	}
}

func TestCancelableRetrierDoesNotCancelTheOperation(t *testing.T) {
	ctx, log := newRetryFixture()
	stop, cancel := context.WithCancel(context.Background())
	cancel()

	retrier := NewCancelableConstantRetrier[int](stop, ctx, log).Build()

	var opCtxErr error
	ret, err := retrier.Retry(func(opCtx context.Context) (int, error) {
		opCtxErr = opCtx.Err()
		return 1, nil
	})
	if opCtxErr != nil {
		t.Fatalf("operation context error = %v, want the stop context not to reach the operation", opCtxErr)
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ret != 1 {
		t.Fatalf("ret = %d, want a successful attempt to win over the stop context", ret)
	}
}

func TestCancelableExponentialRetrierStopsWhileWaiting(t *testing.T) {
	ctx, log := newRetryFixture()
	stop, cancel := context.WithCancel(context.Background())
	defer cancel()

	var attempts int
	retrier := NewCancelableExponentialRetrier[int](stop, ctx, log).
		WithMultiplier(2).
		WithMaxInterval(time.Hour).
		WithMaxElapsedTime(0).
		WithInterval(time.Hour).
		WithMaxRetries(0).
		Build()

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	ret, err := retrier.Retry(func(context.Context) (int, error) {
		attempts++
		return 9, fmt.Errorf("boom")
	})

	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("Retry took %v, want it to end as soon as the stop context is cancelled", elapsed)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want it to wrap context.Canceled", err)
	}
	if ret != 9 {
		t.Fatalf("ret = %d, want the last attempt's value 9", ret)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestCancelableRetrierReportsTheOperationErrorWhenNotStopped(t *testing.T) {
	ctx, log := newRetryFixture()
	stop, cancel := context.WithCancel(context.Background())
	defer cancel()

	retrier := NewCancelableConstantRetrier[int](stop, ctx, log).
		WithInterval(time.Millisecond).
		WithMaxRetries(2).
		Build()

	ret, err := retrier.Retry(func(context.Context) (int, error) {
		return 0, fmt.Errorf("boom")
	})
	if errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the operation error, not a stop", err)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ret != 0 {
		t.Fatalf("ret = %d, want 0", ret)
	}
}

func TestPlainRetrierKeepsAnUninterruptibleInitialWait(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	retrier := NewConstantRetrier[int](ctx, log).
		WithInitialWait(100 * time.Millisecond).
		Build()

	start := time.Now()
	if _, err := retrier.Retry(func(context.Context) (int, error) {
		return 1, nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("initial wait took %v, want the plain retrier to keep sleeping through it", elapsed)
	}
}

func assertInDelta(t *testing.T, want, got, delta time.Duration, msg string) {
	t.Helper()
	if d := time.Duration(math.Abs(float64(got - want))); d > delta {
		t.Fatalf("%s: got %v, want %v (±%v)", msg, got, want, delta)
	}
}
