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

func WithContext(ctx context.Context) Option {
	return func(c *Config) {
		c.ctx = ctx
	}
}

func RestartAfter(d time.Duration) Option {
	return func(c *Config) {
		c.restartDelay = d
	}
}
