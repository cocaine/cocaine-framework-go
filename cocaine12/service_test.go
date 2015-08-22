package cocaine12

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/context"
)

func TestService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped without Cocaine")
	}

	var (
		call, write, first, second uint64
		wg                         sync.WaitGroup
	)

	ctx := context.Background()
	s, err := NewService(ctx, "echo", nil)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			ch, err := s.Call("enqueue", "http")
			if err != nil {
				t.Fatal(err)
			}
			defer ch.Call("close")

			atomic.AddUint64(&call, 1)
			ch.Call("write", []byte("OK"))

			if _, err = ch.Get(ctx); err != nil {
				t.Fatal(err)
			}
			atomic.AddUint64(&write, 1)

			if _, err = ch.Get(ctx); err != nil {
				t.Fatal(err)
			}
			atomic.AddUint64(&first, 1)

			if _, err = ch.Get(ctx); err != nil {
				t.Fatal(err)
			}
			atomic.AddUint64(&second, 1)
		}(i)

	}

	ch := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()

	select {
	case <-ch:
	case <-time.After(time.Second * 5):
		t.Fail()
	}

	t.Logf(`
		CALL %d
		WRITE %d,
		FIRST %d,
		SECOND %d`,
		atomic.LoadUint64(&call),
		atomic.LoadUint64(&write),
		atomic.LoadUint64(&first),
		atomic.LoadUint64(&second))
}
