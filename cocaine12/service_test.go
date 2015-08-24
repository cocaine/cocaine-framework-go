package cocaine12

import (
	"fmt"
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
		call, write, first uint64
		wg                 sync.WaitGroup
	)

	ctx := context.Background()
	s, err := NewService(ctx, "echo", nil)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			ch, err := s.Call(ctx, "enqueue", "ping")
			if err != nil {
				fmt.Println(err)
				t.Fatal(err)
			}
			defer ch.Call("close")
			atomic.AddUint64(&call, 1)

			ch.Call("write", []byte("OK"))
			atomic.AddUint64(&write, 1)

			if _, err = ch.Get(ctx); err != nil {
				fmt.Println(err)
				t.Fatal(err)
			}
			atomic.AddUint64(&first, 1)
		}(i)

	}

	ch := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()

	select {
	case <-ch:
	case <-time.After(time.Second * 10):
		t.Fail()
		panic("give me traceback")
	}

	t.Logf(`
		CALL %d
		WRITE %d,
		FIRST %d`,
		atomic.LoadUint64(&call),
		atomic.LoadUint64(&write),
		atomic.LoadUint64(&first))
}
