package ipc

import (
	"context"
	"fmt"
	"net"
	"time"
)

const (
	// DefaultDialRetryInterval is the pause between dial attempts.
	DefaultDialRetryInterval = 100 * time.Millisecond
	// DefaultDialAttemptTimeout bounds each dial attempt in DialRetry.
	DefaultDialAttemptTimeout = time.Second
)

type dialRetryConfig struct {
	interval       time.Duration
	attemptTimeout time.Duration
}

func defaultDialRetryConfig() dialRetryConfig {
	return dialRetryConfig{
		interval:       DefaultDialRetryInterval,
		attemptTimeout: DefaultDialAttemptTimeout,
	}
}

// DialRetryOption configures DialRetry.
type DialRetryOption func(*dialRetryConfig)

// DialRetryInterval sets the pause between attempts. Non-positive values use DefaultDialRetryInterval.
func DialRetryInterval(d time.Duration) DialRetryOption {
	return func(c *dialRetryConfig) { c.interval = d }
}

// DialRetryAttemptTimeout bounds each dial attempt. Non-positive values use DefaultDialAttemptTimeout.
func DialRetryAttemptTimeout(d time.Duration) DialRetryOption {
	return func(c *dialRetryConfig) { c.attemptTimeout = d }
}

var (
	dialRetryWait = func(ctx context.Context, d time.Duration) error {
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
	dialAttempt = DialTimeout
)

// DialRetry dials addr until success or ctx is canceled.
func DialRetry(ctx context.Context, addr Addr, opts ...DialRetryOption) (net.Conn, error) {
	cfg := defaultDialRetryConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.interval <= 0 {
		cfg.interval = DefaultDialRetryInterval
	}
	if cfg.attemptTimeout <= 0 {
		cfg.attemptTimeout = DefaultDialAttemptTimeout
	}

	var lastErr error
	for {
		conn, err := dialAttempt(ctx, addr, cfg.attemptTimeout)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		if err := dialRetryWait(ctx, cfg.interval); err != nil {
			if lastErr != nil {
				return nil, fmt.Errorf("ipc: dial retry: %w: %w", err, lastErr)
			}
			return nil, err
		}
	}
}
