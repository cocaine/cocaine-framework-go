package cocaine12

import (
	"context"
	"errors"
	"io"
	"syscall"

	"github.com/tinylib/msgp/msgp"
)

type request struct {
	messageTypeDetector
	headers    CocaineHeaders
	fromWorker chan *message
	toHandler  chan *message
	closed     chan struct{}
}

const (
	cworkererrorcategory = 42
	cdefaulterrrorcode   = 100
)

var (
	// ErrStreamIsClosed means that a response stream is closed
	ErrStreamIsClosed = errors.New("Stream is closed")
	// ErrBadPayload means that a message payload is malformed
	ErrBadPayload = errors.New("payload is not []byte")
	// ErrMalformedErrorMessage means that we receive a corrupted or
	// unproper message
	ErrMalformedErrorMessage = &ErrRequest{
		Message:  "malformed error message",
		Category: cworkererrorcategory,
		Code:     cdefaulterrrorcode,
	}
)

func newRequest(mtd messageTypeDetector, headers CocaineHeaders) *request {
	request := &request{
		messageTypeDetector: mtd,
		fromWorker:          make(chan *message),
		toHandler:           make(chan *message),
		closed:              make(chan struct{}),
		headers:             headers,
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

// Headers returns current associated headers. Next Read call
// will override associated headers.
// Multiple calls of the method returns the same shared Headers value.
func (request *request) Headers() CocaineHeaders {
	return request.headers
}

func (request *request) Read(ctx context.Context) ([]byte, error) {
	select {
	// Choke never reaches this select,
	// as it is simulated by closing toHandler channel.
	// So msg can be either Chunk or Error.
	case msg, ok := <-request.toHandler:
		if !ok {
			return nil, ErrStreamIsClosed
		}

		// reset current headers
		request.headers = msg.headers

		if request.isChunk(msg) {
			sz, b, err := msgp.ReadArrayHeaderBytes(msg.payload)
			if err != nil || sz != 1 {
				return nil, ErrBadPayload
			}
			// NOTE: ZeroCopy unpacking
			result, _, err := msgp.ReadBytesZC(b)
			if err != nil {
				return nil, ErrBadPayload
			}
			return result, nil
		}

		var errRequest = new(ErrRequest)
		if _, err := errRequest.UnmarshalMsg(msg.payload); err != nil {
			return nil, err
		}
		return nil, errRequest
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (request *request) push(msg *message) {
	request.fromWorker <- msg
}

func (request *request) Close() {
	close(request.closed)
}

type response struct {
	handlerProtocolGenerator
	session  uint64
	toWorker asyncSender
	closed   bool
}

func newResponse(h handlerProtocolGenerator, session uint64, toWorker asyncSender) *response {
	response := &response{
		handlerProtocolGenerator: h,
		session:                  session,
		toWorker:                 toWorker,
		closed:                   false,
	}

	return response
}

// Write sends chunk of data to a client.
// It copies data to follow io.Writer rule about not retaining a buffer
func (r *response) Write(data []byte) (n int, err error) {
	// According to io.Writer spec
	// I must not retain provided []byte
	var cpy = append([]byte(nil), data...)
	if err := r.ZeroCopyWrite(cpy); err != nil {
		return 0, err
	}

	return len(data), nil
}

// ZeroCopyWrite sends data to a client.
// Response takes the ownership of the buffer, so provided buffer must not be edited.
func (r *response) ZeroCopyWrite(data []byte) error {
	if r.isClosed() {
		return io.ErrClosedPipe
	}

	r.toWorker.Send(r.newChunk(r.session, data))
	return nil
}

// Notify a client about finishing the datastream.
func (r *response) Close() error {
	if r.isClosed() {
		// we treat it as a network connection
		return syscall.EINVAL
	}

	r.close()
	r.toWorker.Send(r.newChoke(r.session))
	return nil
}

// Send error to a client. Specify code and message, which describes this error.
func (r *response) ErrorMsg(code int, message string) error {
	if r.isClosed() {
		return io.ErrClosedPipe
	}

	r.close()
	r.toWorker.Send(r.newError(
		// current session number
		r.session,
		// category
		cworkererrorcategory,
		// error code
		code,
		// error message
		message,
	))
	return nil
}

func (r *response) close() {
	r.closed = true
}

func (r *response) isClosed() bool {
	return r.closed
}

func loop(input <-chan *message, output chan *message, onclose <-chan struct{}) {
	defer close(output)

	var (
		pending []*message
		closed  = onclose
	)

	for {
		var (
			out   chan *message
			first *message
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
