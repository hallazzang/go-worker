package worker

import (
	"context"
	"time"
)

type Config struct {
	ctx          context.Context
	restartDelay time.Duration
}

type Option func(*Config)

// WithContext sets parent context of workers.
func WithContext(ctx context.Context) Option {
	return func(c *Config) {
		c.ctx = ctx
	}
}

// RestartAfter sets worker restart delay.
func RestartAfter(d time.Duration) Option {
	return func(c *Config) {
		c.restartDelay = d
	}
}
