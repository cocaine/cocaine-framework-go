package cocaine12

import (
	// "fmt"
	"sync"
	"sync/atomic"
	// "time"
)

type Channel interface {
	Rx
	Tx
}

type Rx interface {
	Get() (ServiceResult, error)
	// GetWithTimeout(timeout time.Duration) (ServiceResult, error)
	Push(ServiceResult)
}

type Tx interface {
	// Call(name string, args ...interface{}) error
}

type channel struct {
	rx
	tx
}

type rx struct {
	pushBuffer chan ServiceResult

	sync.Mutex
	queue   []ServiceResult
	pending int32
}

func (rx *rx) enqueue() {
	atomic.AddInt32(&(rx.pending), 1)
}

func (rx *rx) dequeue() {
	atomic.AddInt32(&(rx.pending), -1)
}

func (rx *rx) ispending() bool {
	return atomic.LoadInt32(&(rx.pending)) > 0
}

func (rx *rx) Get() (ServiceResult, error) {
	var res ServiceResult
	rx.Lock()
	if len(rx.queue) > 0 {
		res = rx.queue[0]
		rx.queue = rx.queue[1:]
	}
	rx.enqueue()
	rx.Unlock()
	res = <-rx.pushBuffer
	return res, nil
}

// func (rx *rx) GetWithTimeout(timeout time.Duration) (ServiceResult, error) {
// 	rx.enqueugetter()
// 	select {
// 	case res := <-rx.pollBuffer:
// 		return res, nil
// 	case <-time.After(timeout):
// 		return nil, fmt.Errorf("Timeout error")
// 	}
// }

func (rx *rx) Push(res ServiceResult) {
	// fast path
	select {
	case rx.pushBuffer <- res:
		rx.dequeue()
	default:
	}

	rx.Lock()
	defer rx.Unlock()
	if !rx.ispending() {
		rx.pushBuffer <- res
		rx.dequeue()
		return
	}

	rx.queue = append(rx.queue, res)
}

type tx struct {
	service *Service
	id      uint64
	txChan  chan ServiceResult
}
