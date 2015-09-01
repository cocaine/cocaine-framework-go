package cocaine12

import (
	"fmt"
	"sync"

	"golang.org/x/net/context"
)

type Channel interface {
	Rx
	Tx
}

type Rx interface {
	Get(context.Context) (ServiceResult, error)
	push(ServiceResult)
}

type Tx interface {
	Call(ctx context.Context, name string, args ...interface{}) error
}

type channel struct {
	rx
	tx
}

type rx struct {
	pushBuffer chan ServiceResult
	rxTree     *streamDescription

	sync.Mutex
	queue []ServiceResult
	done  bool
}

func (rx *rx) Get(ctx context.Context) (ServiceResult, error) {
	if rx.done {
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

func (rx *rx) push(res ServiceResult) {
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
	txTree  *streamDescription
	id      uint64
	done    bool
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
		CommonMessageInfo{
			tx.id,
			method},
		args,
	}

	tx.service.sendMsg(msg)
	return nil
}
