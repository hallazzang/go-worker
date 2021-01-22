package worker

import (
	"context"
	"sync"
	"time"
)

type Pool struct {
	ctx          context.Context
	cancel       context.CancelFunc
	f            func(context.Context)
	restartDelay time.Duration
	workers      []context.CancelFunc
	mux          sync.Mutex
	wg           sync.WaitGroup
}

// NewPool creates a new pool of given worker function.
// Since it starts with zero workers, the pool does not run workers
// until the first call of Resize.
func NewPool(f func(context.Context), options ...Option) *Pool {
	c := Config{
		ctx:          context.Background(),
		restartDelay: 0,
	}
	for _, o := range options {
		o(&c)
	}
	ctx, cancel := context.WithCancel(c.ctx)
	return &Pool{
		ctx:          ctx,
		cancel:       cancel,
		f:            f,
		restartDelay: c.restartDelay,
	}
}

// Resize changes active worker count.
// If given n is greater than current worker count,
// it starts new workers. If given n is less than current
// worker count, it cancels running workers' context.
// This method is concurrent-safe.
func (p *Pool) Resize(n int) {
	p.mux.Lock()
	defer p.mux.Unlock()
	if l := len(p.workers); l < n {
		p.wg.Add(n - l)
		for i := 0; i < n-l; i++ {
			ctx, cancel := context.WithCancel(p.ctx)
			p.workers = append(p.workers, cancel)
			go p.startWorker(ctx)
		}
	} else if l > n {
		for i := 0; i < l-n; i++ {
			p.workers[i]()
		}
		p.workers = p.workers[l-n:]
	}
}

// Close cancels all running workers' context and wait
// for all workers to stop.
func (p *Pool) Close() {
	p.cancel()
	p.wg.Wait()
}

func (p *Pool) startWorker(ctx context.Context) {
	defer p.wg.Done()
	for {
		p.f(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(p.restartDelay):
		}
	}
}
