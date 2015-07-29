package trace

import (
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"
)

func a(ctx context.Context, i int) {
	_, done := WithTrace(ctx)
	defer done("end %d", i)

	time.Sleep(200 * time.Millisecond)
}

func b(ctx context.Context) {
	ctx, done := WithTrace(ctx)
	defer done("end b")
	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			a(ctx, i)
		}(i)
	}

	wg.Wait()
}

func TestTrace(t *testing.T) {
	tr := NewTraced()
	b(tr)
}
