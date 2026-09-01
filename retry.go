package utils

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cenkalti/backoff"
)

// TODO alias to backoff.Permanent?

// RetriableOp defines an operation that can be retriable. The provided context is the same one used by the
// underlying backoff.RetryNotify() call so it must be used within the operation to make sure things are in sync
type RetriableOp[T any] func(ctx context.Context) (T, error)

// Retrier is helper interface with a Retry method which is a wrapper to backoff.RetryNotify to make our lives easier
type Retrier[T any] interface {
	// Retry retries the op() function following the retry strategy defined by the builder. It returns exactly what
	// op() returns
	Retry(op RetriableOp[T]) (T, error)
	Reset()
}

type retrier[T any] struct {
	config  retrybuilder[T]
	backoff backoff.BackOff
}

func newRetrier[T any](config retrybuilder[T], b backoff.BackOff) *retrier[T] {
	return &retrier[T]{
		config:  config,
		backoff: backoff.WithMaxRetries(b, config.maxRetries),
	}
}

func (c *retrier[T]) Retry(op RetriableOp[T]) (result T, err error) {
	ctx, cancel := c.config.context()
	defer cancel()

	stop := c.waitOrStop(ctx, c.config.wait)

	operation := func() (err error) {
		result, err = op(ctx)
		return err
	}
	err = backoff.RetryNotify(
		operation,
		backoff.WithContext(c.backoff, stop),
		func(err error, d time.Duration) {
			fields := append(c.config.fields, "error", err, slog.String("wait", d.String()))
			c.config.logger.Log(ctx, c.config.lvl, "retrying operation", fields...)
		},
	)
	// backoff always attempts once, so result holds a real value. Report the stop so the caller can
	// tell a shutdown from a genuine failure.
	if err != nil && c.config.stop != nil && c.config.stop.Err() != nil {
		return result, fmt.Errorf("retry stopped: %w", c.config.stop.Err())
	}
	return result, err
}

// waitOrStop waits for d and returns the context governing the waits between attempts. A plain
// retrier sleeps through d and is governed by ctx, which is where WithContextTimeout takes effect;
// a cancelable one cuts d short once its stop context is done, and is governed by that.
func (c *retrier[T]) waitOrStop(ctx context.Context, d time.Duration) context.Context {
	if c.config.stop == nil {
		time.Sleep(d)
		return ctx
	}
	if d > 0 {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-c.config.stop.Done():
		case <-t.C:
		}
	}
	return c.config.stop
}

func (c *retrier[T]) Reset() {
	c.backoff.Reset()
}

type internalBuilder[T any] interface {
	// Build builds and return the concrete retrier
	Build() Retrier[T]
}

// RetryBuilder is a simple builder interface to configure a Retry object
type RetryBuilder[T any] interface {
	internalBuilder[T]

	// WithMaxRetries sets the max amount of retries before bailing. It wraps backoff.WithMaxRetries()
	WithMaxRetries(retries uint64) RetryBuilder[T]

	// WithInterval sets the initial interval. Itss usage is implementation specific: constant retrier uses it as
	// its constante interval, exponential retrier uses it as the initial interval
	WithInterval(interval time.Duration) RetryBuilder[T]

	// WithLogging sets the log level to be used when notifying that an operation is retried
	WithLogging(level slog.Level) RetryBuilder[T]

	// WithLogFields adds these fields to the logging message printed when the operation is retried
	WithLogFields(fields ...any) RetryBuilder[T]

	// WithContextTimeout sets the context timeout. If set, the retrier will create a child context with this timeout
	// Otherwise, it will use the provided ctx directly.
	WithContextTimeout(timeout time.Duration) RetryBuilder[T]

	// WithInitialWait sets the initial wait before starting to retry the operation. This can be useful in cases
	// where it's known that the first try will fail such as fetching a transaction receipt
	WithInitialWait(wait time.Duration) RetryBuilder[T]
}

type retrybuilder[T any] struct {
	ctx        context.Context
	stop       context.Context
	maxRetries uint64
	interval   time.Duration
	logger     *slog.Logger
	lvl        slog.Level
	fields     []any
	timeout    time.Duration
	wait       time.Duration
	builder    internalBuilder[T]
}

func newRetryBuilder[T any](ctx context.Context, builder internalBuilder[T], logger *slog.Logger) retrybuilder[T] {
	return retrybuilder[T]{
		builder:    builder,
		ctx:        ctx,
		logger:     logger,
		maxRetries: 10,
		interval:   2 * time.Second,
		lvl:        slog.LevelWarn,
	}
}

func (c *retrybuilder[T]) context() (context.Context, context.CancelFunc) {
	if c.timeout == 0 {
		return c.ctx, func() {}
	}
	return context.WithTimeout(c.ctx, c.timeout)
}

func (c *retrybuilder[T]) WithInitialWait(wait time.Duration) RetryBuilder[T] {
	c.wait = wait
	return c
}

func (c *retrybuilder[T]) WithContextTimeout(timeout time.Duration) RetryBuilder[T] {
	c.timeout = timeout
	return c
}

func (c *retrybuilder[T]) WithLogFields(fields ...any) RetryBuilder[T] {
	c.fields = fields
	return c
}

func (c *retrybuilder[T]) WithLogging(level slog.Level) RetryBuilder[T] {
	c.lvl = level
	return c
}

func (c *retrybuilder[T]) WithMaxRetries(retries uint64) RetryBuilder[T] {
	c.maxRetries = retries
	return c
}

func (c *retrybuilder[T]) WithInterval(interval time.Duration) RetryBuilder[T] {
	c.interval = interval
	return c
}

func (c *retrybuilder[T]) Build() Retrier[T] {
	return c.builder.Build()
}

type constant[T any] struct {
	retrybuilder[T]
}

// NewConstantRetrier builder of a ConstantRetrier which is a wrapper of backoff.ConstantBackOff
func NewConstantRetrier[T any](ctx context.Context, logger *slog.Logger) RetryBuilder[T] {
	return newConstant[T](ctx, logger)
}

// NewCancelableConstantRetrier builds a ConstantRetrier whose waits end as soon as stop is done.
// See NewCancelableExponentialRetrier for the contract.
func NewCancelableConstantRetrier[T any](stop, ctx context.Context, logger *slog.Logger) RetryBuilder[T] {
	c := newConstant[T](ctx, logger)
	c.stop = stop
	return c
}

func newConstant[T any](ctx context.Context, logger *slog.Logger) *constant[T] {
	c := &constant[T]{}
	c.retrybuilder = newRetryBuilder(ctx, c, logger)
	return c
}

// Build implements the RetryBuilder interface
func (c *constant[T]) Build() Retrier[T] {
	return newRetrier(c.retrybuilder, backoff.NewConstantBackOff(c.interval))
}

type exponential[T any] struct {
	retrybuilder[T]
	multiplier  float64
	maxInterval time.Duration
	maxElapsed  time.Duration
}

// ExponentialRetryBuilder extends RetryBuilder with exponential-specific configuration options
type ExponentialRetryBuilder[T any] interface {
	RetryBuilder[T]
	// WithMultiplier sets the multiplier for exponential backoff (default is 1.5)
	WithMultiplier(multiplier float64) ExponentialRetryBuilder[T]
	// WithMaxInterval sets the maximum interval between retries (default is 60s)
	WithMaxInterval(maxInterval time.Duration) ExponentialRetryBuilder[T]
	// WithMaxElapsedTime sets the maximum elapsed time for retries (default is 15min)
	// set to zero to disable
	WithMaxElapsedTime(maxElapsedTime time.Duration) ExponentialRetryBuilder[T]
}

// NewExponentialRetrier builder of an ExponentialRetrier which is a wrapper of backoff.ExponentialBackOff
func NewExponentialRetrier[T any](ctx context.Context, logger *slog.Logger) ExponentialRetryBuilder[T] {
	return newExponential[T](ctx, logger)
}

// NewCancelableExponentialRetrier builds an ExponentialRetrier whose waits end as soon as stop is
// done, so a long backoff interval never holds up a shutdown. stop never reaches the operation, so
// an attempt that has begun always finishes and the operation runs at least once. On stop, Retry
// returns the last attempt's value and an error wrapping stop.Err(); a successful attempt still
// wins. Does not compose with WithContextTimeout.
func NewCancelableExponentialRetrier[T any](stop, ctx context.Context, logger *slog.Logger) ExponentialRetryBuilder[T] {
	c := newExponential[T](ctx, logger)
	c.stop = stop
	return c
}

func newExponential[T any](ctx context.Context, logger *slog.Logger) *exponential[T] {
	c := &exponential[T]{
		multiplier:  backoff.DefaultMultiplier,
		maxInterval: backoff.DefaultMaxInterval,
		maxElapsed:  backoff.DefaultMaxElapsedTime,
	}
	c.retrybuilder = newRetryBuilder(ctx, c, logger)
	return c
}

// WithMultiplier sets the multiplier for exponential backoff
func (c *exponential[T]) WithMultiplier(multiplier float64) ExponentialRetryBuilder[T] {
	c.multiplier = multiplier
	return c
}

// WithMaxInterval sets the maximum interval between retries
func (c *exponential[T]) WithMaxInterval(maxInterval time.Duration) ExponentialRetryBuilder[T] {
	c.maxInterval = maxInterval
	return c
}

// WithMaxElapsedTime sets the maximum elapsed time for retries
func (c *exponential[T]) WithMaxElapsedTime(maxElapsedTime time.Duration) ExponentialRetryBuilder[T] {
	c.maxElapsed = maxElapsedTime
	return c
}

// Build implements the RetryBuilder interface
func (c *exponential[T]) Build() Retrier[T] {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = c.interval
	b.Multiplier = c.multiplier
	b.MaxInterval = c.maxInterval
	b.MaxElapsedTime = c.maxElapsed
	// We don't want any randomization, just pure exponential
	b.RandomizationFactor = 0
	return newRetrier(c.retrybuilder, b)
}
