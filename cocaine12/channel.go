package cocaine12

import (
	"fmt"
	"sync"
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
	Call(name string, args ...interface{}) error
}

type channel struct {
	rx
	tx
}

type rx struct {
	pushBuffer chan ServiceResult
	rxTree     *StreamDescription

	sync.Mutex
	queue []ServiceResult
}

func (rx *rx) Get() (ServiceResult, error) {
	var res ServiceResult

	// fast path
	select {
	case res = <-rx.pushBuffer:
		return res, nil
	default:
	}

	rx.Lock()
	if len(rx.queue) > 0 {
		res = rx.queue[0]
		select {
		case rx.pushBuffer <- res:
			rx.queue = rx.queue[1:]
		default:
		}
	}
	rx.Unlock()
	res = <-rx.pushBuffer
	return res, nil
}

func (rx *rx) Push(res ServiceResult) {
	rx.Lock()
	rx.queue = append(rx.queue, res)
	select {
	case rx.pushBuffer <- rx.queue[0]:
		rx.queue = rx.queue[1:]
	default:
	}
	rx.Unlock()
}

type tx struct {
	service *Service
	txTree  *StreamDescription
	id      uint64
	done    bool
}

func (tx *tx) Call(name string, args ...interface{}) error {
	if tx.done {
		return fmt.Errorf("tx is done")
	}

	method, err := tx.txTree.MethodByName(name)
	if err != nil {
		return err
	}

	z := *(tx.txTree)
	temp := z[method]
	switch temp.StreamDescription {
	case EmptyDescription:
		tx.done = true
	case RecursiveDescription:
		//pass
	default:
		tx.txTree = temp.StreamDescription
	}

	msg := &Message{
		CommonMessageInfo{
			tx.id,
			method},
		args,
	}

	tx.service.sendMsg(msg)
	return nil
}
