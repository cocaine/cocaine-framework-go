package cocaine12

import (
	"errors"
	"fmt"
	"time"
)

type request struct {
	messageTypeDetector
	fromWorker chan *Message
	toHandler  chan *Message
	closed     chan struct{}
}

var (
	// ErrStreamIsClosed means that a response stream is closed
	ErrStreamIsClosed = &ClosedError{}
	// ErrTimeout means that a request is timeouted
	ErrTimeout = &TimeoutError{}
	// ErrBadPayload means that a message payload is malformed
	ErrBadPayload = errors.New("payload is not []byte")
)

type TimeoutError struct{}

func (t *TimeoutError) Error() string { return "TimeoutError" }

type ClosedError struct{}

func (t *ClosedError) Error() string { return "Stream is closed" }

func IsTimeout(err error) bool {
	switch err.(type) {
	case *TimeoutError:
		return true
	}
	return false
}

func newRequest(mtd messageTypeDetector) *request {
	request := &request{
		messageTypeDetector: mtd,
		fromWorker:          make(chan *Message),
		toHandler:           make(chan *Message),
		closed:              make(chan struct{}),
	}

	go loop(
		// input
		request.fromWorker,
		// output
		request.toHandler,
		// onclose
		request.closed,
	)

	return request
}

// ToDo: context?
func (request *request) Read(timeout ...time.Duration) ([]byte, error) {
	var onTimeout <-chan time.Time

	if len(timeout) > 0 {
		onTimeout = time.After(timeout[0])
	}

	select {
	case msg, ok := <-request.toHandler:
		if !ok {
			return nil, ErrStreamIsClosed
		}

		if request.isChunk(msg) {
			if result, isByte := msg.Payload[0].([]byte); isByte {
				return result, nil
			}
			return nil, ErrBadPayload
		} else {
			return nil, fmt.Errorf("error %d: %s", msg.Payload...)
		}
	case <-onTimeout:
		return nil, ErrTimeout
	}
}

func (request *request) push(msg *Message) {
	request.fromWorker <- msg
}

func (request *request) Close() {
	close(request.closed)
}

type response struct {
	handlerProtocolGenerator
	session     uint64
	fromHandler chan *Message
	toWorker    chan *Message
	closed      chan struct{}
}

func newResponse(h handlerProtocolGenerator, session uint64, toWorker chan *Message) *response {
	response := &response{
		handlerProtocolGenerator: h,
		session:                  session,
		fromHandler:              make(chan *Message),
		toWorker:                 toWorker,
		closed:                   make(chan struct{}),
	}

	go loop(
		// input
		response.fromHandler,
		// output
		response.toWorker,
		// onclose
		response.closed,
	)

	return response
}

// Sends chunk of data to a client.
func (r *response) Write(data []byte) {
	if r.isClosed() {
		return
	}

	r.fromHandler <- r.newChunk(r.session, data)
}

// Notify a client about finishing the datastream.
func (r *response) Close() {
	if r.isClosed() {
		return
	}

	r.fromHandler <- r.newChoke(r.session)
	close(r.closed)
}

// Send error to a client. Specify code and message, which describes this error.
func (r *response) ErrorMsg(code int, message string) {
	if r.isClosed() {
		return
	}

	r.fromHandler <- r.newError(
		// current session number
		r.session,
		// error code
		code,
		// error message
		message,
	)

	r.Close()
}

func (r *response) isClosed() bool {
	select {
	case <-r.closed:
		return true
	default:
	}
	return false
}

func loop(input <-chan *Message, output chan *Message, onclose <-chan struct{}) {
	var (
		pending []*Message
		closed  = onclose
	)

	for {
		var (
			out   chan *Message
			first *Message
		)

		if len(pending) > 0 {
			// if we have data to send,
			// pick the first element from the queue
			// and unlock `out case` in select
			// Othrewise `out` is nil
			first = pending[0]
			out = output
		} else if closed == nil {
			// Pending queue is empty
			// and there will be no incoming data
			// as request is closed
			return
		}

		select {
		case incoming := <-input:
			pending = append(pending, incoming)

		case out <- first:
			// help GC a bit
			pending[0] = nil
			// it should be done
			// without memory copy/allocate
			pending = pending[1:]

		case <-closed:
			// It will be triggered on
			// the next iteration as closed is closed
			closed = nil
		}
	}
}
