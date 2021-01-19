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
