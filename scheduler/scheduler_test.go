package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestScheduler_Every(t *testing.T) {
	var mu sync.Mutex
	calls := 0

	s := New(nil, nil, nil)
	s.Every("test", 50*time.Millisecond, func(ctx context.Context) error {
		mu.Lock()
		calls++
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()

	<-done

	mu.Lock()
	if calls < 2 {
		t.Errorf("expected at least 2 calls, got %d", calls)
	}
	mu.Unlock()
}