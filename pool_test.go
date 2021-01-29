package worker

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestPool(t *testing.T) {
	defer goleak.VerifyNone(t)

	p := NewPool(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(rand.Intn(500)) * time.Millisecond):
			}
		}
	})
	defer p.Close()

	p.Resize(3)
	time.Sleep(50 * time.Millisecond)
	if len(p.WorkerIDs()) != 3 {
		t.Fatal("worker number should be 3")
	}

	p.Resize(1000)
	time.Sleep(50 * time.Millisecond)
	if len(p.WorkerIDs()) != 1000 {
		t.Fatal("worker number should be 1000")
	}

	p.Resize(1)
	time.Sleep(50 * time.Millisecond)
	if len(p.WorkerIDs()) >= 1000 {
		t.Fatal("worker number should start decreasing")
	}
	time.Sleep(550 * time.Millisecond)
	if len(p.WorkerIDs()) != 1 {
		t.Fatal("worker number should be 1")
	}

	ids := p.WorkerIDs()
	id := ids[0]
	p.KillWorker(id)
	time.Sleep(50 * time.Millisecond)
	ids = p.WorkerIDs()
	if len(ids) != 1 {
		t.Fatal("worker number should be 1")
	}
	if ids[0] == id {
		t.Fatal("worker should have been killed")
	}

	p.Resize(10)
	prevIDs := make(map[string]struct{})
	for _, id := range p.WorkerIDs() {
		prevIDs[id] = struct{}{}
		p.KillWorker(id)
	}
	time.Sleep(50 * time.Millisecond)
	ids = p.WorkerIDs()
	if len(ids) != 10 {
		t.Fatal("worker number should be 10")
	}
	for _, id := range ids {
		if _, ok := prevIDs[id]; ok {
			t.Fatal("worker should have been killed")
		}
	}
}
