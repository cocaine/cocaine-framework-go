package cocaine12

import (
	"context"
	"fmt"
	"sync"
)

type Channel interface {
	Rx
	Tx
}

type Rx interface {
	Get(context.Context) (ServiceResult, error)
	Closed() bool
	push(ServiceResult)
}

type Tx interface {
	Call(ctx context.Context, name string, args ...interface{}) error
}

type channel struct {
	// we call when data frame arrives
	traceReceived CloseSpan
	// we call when data is sent
	traceSent CloseSpan

	rx
	tx
}

func (ch *channel) push(res ServiceResult) {
	ch.traceReceived()
	ch.rx.push(res)
}

func (ch *channel) Call(ctx context.Context, name string, args ...interface{}) error {
	ch.traceSent()
	return ch.tx.Call(ctx, name, args...)
}

type rx struct {
	service    *Service
	pushBuffer chan ServiceResult
	rxTree     *streamDescription
	id         uint64

	sync.Mutex
	queue []ServiceResult
	done  bool
}

func (rx *rx) Get(ctx context.Context) (ServiceResult, error) {
	if rx.Closed() {
		return nil, ErrStreamIsClosed
	}

	var res ServiceResult

	// fast path
	select {
	case res = <-rx.pushBuffer:
	default:
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

		select {
		case res = <-rx.pushBuffer:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	treeMap := *(rx.rxTree)
	method, _, _ := res.Result()
	temp := treeMap[method]

	switch temp.Description.Type() {
	case emptyDispatch:
		rx.done = true
	case recursiveDispatch:
		// pass
	case otherDispatch:
		rx.rxTree = temp.Description
	}

	// allow to attach various protocols
	switch temp.Name {
	case "error":
		var (
			catAndCode [2]int
			message    string
		)

		if err := res.ExtractTuple(&catAndCode, &message); err != nil {
			return res, err
		}

		res.setError(&ErrRequest{
			Message:  message,
			Category: catAndCode[0],
			Code:     catAndCode[1],
		})
	}

	return res, nil
}

func (rx *rx) Closed() bool {
	return rx.done
}

func (rx *rx) push(res ServiceResult) {
	rx.Lock()
	rx.queue = append(rx.queue, res)
	select {
	case rx.pushBuffer <- rx.queue[0]:
		rx.queue = rx.queue[1:]
	default:
	}
	rx.Unlock()

	treeMap := *(rx.rxTree)
	method, _, _ := res.Result()
	if temp := treeMap[method]; temp.Description.Type() == emptyDispatch {
		rx.service.sessions.Detach(rx.id)
	}
}

type tx struct {
	service *Service
	txTree  *streamDescription
	id      uint64
	done    bool

	headers CocaineHeaders
}

func (tx *tx) Call(ctx context.Context, name string, args ...interface{}) error {
	if tx.done {
		return fmt.Errorf("tx is done")
	}

	method, err := tx.txTree.MethodByName(name)
	if err != nil {
		return err
	}

	treeMap := *(tx.txTree)
	temp := treeMap[method]

	switch temp.Description.Type() {
	case emptyDispatch:
		tx.done = true

	case recursiveDispatch:
		//pass

	case otherDispatch:
		tx.txTree = temp.Description
	}

	msg := &Message{
		CommonMessageInfo: CommonMessageInfo{tx.id, method},
		Payload:           args,
		Headers:           tx.headers,
	}

	tx.service.sendMsg(msg)
	return nil
}
