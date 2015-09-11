package cocaine12

import (
	"errors"
	"io"
	"syscall"

	"golang.org/x/net/context"
)

type request struct {
	messageTypeDetector
	fromWorker chan *Message
	toHandler  chan *Message
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

func (request *request) Read(ctx context.Context) ([]byte, error) {
	select {
	// Choke never reaches this select,
	// as it is simulated by closing toHandler channel.
	// So msg can be either Chunk or Error.
	case msg, ok := <-request.toHandler:
		if !ok {
			return nil, ErrStreamIsClosed
		}

		if request.isChunk(msg) {
			if result, isByte := msg.Payload[0].([]byte); isByte {
				return result, nil
			}
			return nil, ErrBadPayload
		}

		// Error message
		if len(msg.Payload) == 0 {
			return nil, ErrMalformedErrorMessage
		}

		var perr struct {
			CodeInfo [2]int
			Message  string
		}

		if err := convertPayload(msg.Payload, &perr); err != nil {
			return nil, err
		}

		return nil, &ErrRequest{
			Message:  perr.Message,
			Category: perr.CodeInfo[0],
			Code:     perr.CodeInfo[1],
		}
	case <-ctx.Done():
		return nil, ctx.Err()
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

// Sends chunk of data to a client.
func (r *response) Write(data []byte) (n int, err error) {
	if r.isClosed() {
		return 0, io.ErrClosedPipe
	}

	r.toWorker.Send(r.newChunk(r.session, data))
	return len(data), nil
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

func loop(input <-chan *Message, output chan *Message, onclose <-chan struct{}) {
	defer close(output)

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
