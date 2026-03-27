package retry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"uasl-reservation/internal/pkg/logger"
)

const (
	DefaultMaxAttempts = 3
	DefaultBaseDelay   = 200 * time.Millisecond
)

type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
	ShouldRetry func(error) bool
}

func DefaultConfig() Config {
	return Config{
		MaxAttempts: DefaultMaxAttempts,
		BaseDelay:   DefaultBaseDelay,
		ShouldRetry: DefaultShouldRetry,
	}
}

func WithBackoff(
	ctx context.Context,
	operation func(context.Context) error,
	config Config,
) error {

	if config.MaxAttempts <= 0 {
		config.MaxAttempts = DefaultMaxAttempts
	}
	if config.BaseDelay <= 0 {
		config.BaseDelay = DefaultBaseDelay
	}
	if config.ShouldRetry == nil {
		config.ShouldRetry = DefaultShouldRetry
	}

	var lastErr error
	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {

		if ctx.Err() != nil {
			return ctx.Err()
		}

		lastErr = operation(ctx)
		if lastErr == nil {
			return nil
		}

		if !config.ShouldRetry(lastErr) {
			logger.LogInfo("retry: non-retryable error, stopping retry attempts",
				"attempt", attempt,
				"error", lastErr.Error(),
			)
			return lastErr
		}

		if attempt == config.MaxAttempts {
			logger.LogInfo("retry: max attempts reached",
				"max_attempts", config.MaxAttempts,
				"error", lastErr.Error(),
			)
			break
		}

		sleepDuration := config.BaseDelay * time.Duration(1<<(attempt-1))
		logger.LogInfo("retry: operation failed, retrying after backoff",
			"attempt", attempt,
			"max_attempts", config.MaxAttempts,
			"sleep_duration", sleepDuration.String(),
			"error", lastErr.Error(),
		)

		select {
		case <-time.After(sleepDuration):

		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return lastErr
}

func DefaultShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	type statusCoder interface{ StatusCode() int }
	var sc statusCoder
	if errors.As(err, &sc) {
		code := sc.StatusCode()

		if code >= 500 || code == 429 {
			return true
		}

		return false
	}

	type temporaryError interface {
		Temporary() bool
	}
	type timeoutError interface {
		Timeout() bool
	}

	var tempErr temporaryError
	var timeErr timeoutError

	if errors.As(err, &tempErr) && tempErr.Temporary() {
		return true
	}

	if errors.As(err, &timeErr) && timeErr.Timeout() {
		return true
	}

	logger.LogInfo("retry: non-retryable error encountered",
		"error_type", fmt.Sprintf("%T", err),
		"error", err.Error(),
	)
	return false
}
