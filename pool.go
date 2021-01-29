package worker

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
)

type (
	IDFunc func() string
)

type Pool struct {
	ctx          context.Context
	cancel       context.CancelFunc
	f            func(context.Context)
	restartDelay time.Duration
	workers      []*worker
	workerMap    map[string]*worker
	idFunc       IDFunc
	mux          sync.Mutex
	wg           sync.WaitGroup
}

type worker struct {
	id     string
	cancel context.CancelFunc
	closed chan struct{}
}

func DefaultIDFunc() string {
	return hex.EncodeToString(uuid.NewV4().Bytes())
}

// NewPool creates a new pool of given worker function.
// Since it starts with zero workers, the pool does not run workers
// until the first call of Resize.
// Workers can retrieve their id from the context using IDFromContext.
func NewPool(f func(context.Context), options ...Option) *Pool {
	c := Config{
		ctx:          context.Background(),
		restartDelay: 0,
		idFunc:       DefaultIDFunc,
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
		workerMap:    make(map[string]*worker),
		idFunc:       c.idFunc,
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
		for i := 0; i < n-l; i++ {
			p.startWorker()
		}
	} else if l > n {
		for i := 0; i < l-n; i++ {
			p.workers[i].cancel()
			delete(p.workerMap, p.workers[i].id)
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

// WorkerIDs returns all worker ids.
func (p *Pool) WorkerIDs() []string {
	p.mux.Lock()
	defer p.mux.Unlock()
	var ids []string
	for _, w := range p.workers {
		ids = append(ids, w.id)
	}
	return ids
}

// KillWorker kills the worker with given id, causing the pool to
// respawn a new worker.
// KillWorker returns false if there was no such worker.
func (p *Pool) KillWorker(id string) bool {
	p.mux.Lock()
	defer p.mux.Unlock()
	if w, ok := p.workerMap[id]; ok {
		w.cancel()
		delete(p.workerMap, id)
		for i := range p.workers {
			if p.workers[i].id == id {
				p.workers = append(p.workers[:i], p.workers[i+1:]...)
				break
			}
		}
		go func() {
			<-w.closed
			p.mux.Lock()
			defer p.mux.Unlock()
			p.startWorker()
		}()
		return true
	}
	return false
}

func (p *Pool) startWorker() {
	var id string
	for {
		id = p.idFunc()
		if _, ok := p.workerMap[id]; !ok {
			break
		}
	}
	ctx, cancel := context.WithCancel(p.ctx)
	closed := make(chan struct{})
	w := &worker{id, cancel, closed}
	p.workers = append(p.workers, w)
	p.workerMap[id] = w
	p.wg.Add(1)
	go p.runWorker(contextWithID(ctx, id), closed)
}

func (p *Pool) runWorker(ctx context.Context, closed chan<- struct{}) {
	defer p.wg.Done()
	defer close(closed)
	for {
		p.f(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(p.restartDelay):
		}
	}
}
