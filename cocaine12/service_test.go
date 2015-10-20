package cocaine12

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
			defer ch.Call(ctx, "close")
			atomic.AddUint64(&call, 1)

			ch.Call(ctx, "write", []byte("OK"))
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

func TestDisconnectedError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped without Cocaine")
	}

	ctx := context.Background()

	s, err := NewService(ctx, "locator", nil)
	if err != nil {
		t.Fatal(err)
	}

	// passing wrong arguments leads to disconnect
	ch, err := s.Call(ctx, "resolve", 1, 2, 3, 4, 5, 6)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ch.Get(ctx)
	assert.EqualError(t, err, "Disconnected")
}

func TestReconnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped without Cocaine")
	}

	ctx := context.Background()

	s, err := NewService(ctx, "locator", nil)
	if err != nil {
		t.Fatal(err)
	}

	// passing wrong arguments leads to disconnect
	ch, err := s.Call(ctx, "resolve", 1, 2, 3, 4, 5, 6)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ch.Get(ctx)
	assert.EqualError(t, err, "Disconnected")

	_, err = s.Call(ctx, "resolve", 1, 2, 3, 4, 5, 6)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTimeoutError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped without Cocaine")
	}

	ctx := context.Background()
	s, err := NewService(ctx, "locator", nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ = context.WithTimeout(context.Background(), time.Microsecond*5)
	// passing wrong arguments leads to disconnect
	ch, err := s.Call(ctx, "resolve", "locator")
	if err != nil {
		t.Fatal(err)
	}

	_, err = ch.Get(ctx)
	if !assert.Error(t, ctx.Err()) {
		t.FailNow()
	}
	assert.EqualError(t, err, ctx.Err().Error())
}

func TestRxClosedGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped without Cocaine")
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	s, err := NewService(ctx, "locator", nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range s.ServiceInfo.API {
		t.Logf("%s %v %v\n", v.Name, v.Downstream, v.Upstream)
		for k, j := range *v.Upstream {
			t.Logf("%v %v\n", k, j)
		}
	}

	// passing wrong arguments leads to disconnect
	ch, err := s.Call(ctx, "connect", 1111)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ch.Get(ctx)
	_, err = ch.Get(ctx)
	assert.EqualError(t, err, ErrStreamIsClosed.Error())
}
