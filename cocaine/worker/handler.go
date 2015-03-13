package worker

import (
	"github.com/hashicorp/go-msgpack/codec"

	"github.com/cocaine/cocaine-framework-go/cocaine/asio"
)

var (
	mh codec.MsgpackHandle
	h  = &mh
)

func loop(input <-chan *asio.Message, output chan *asio.Message, onclose <-chan struct{}) {
	var (
		pending []*asio.Message
		closed  <-chan struct{} = onclose
	)

	for {
		var (
			out   chan *asio.Message
			first *asio.Message
		)

		if len(pending) > 0 {
			// if we have data to send
			// pick first element from the queue
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

type request struct {
	fromWorker chan *asio.Message
	toHandler  chan *asio.Message
	closed     chan struct{}
}

func newRequest() *request {
	request := &request{
		fromWorker: make(chan *asio.Message),
		toHandler:  make(chan *asio.Message),
		closed:     make(chan struct{}),
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

func (request *request) Read() chan *asio.Message {
	return request.toHandler
}

func (request *request) Push(msg *asio.Message) {
	request.fromWorker <- msg
}

func (request *request) Close() {
	close(request.closed)
}

type response struct {
	session     uint64
	fromHandler chan *asio.Message
	toWorker    chan *asio.Message
	closed      chan struct{}
}

func newResponse(session uint64, toWorker chan *asio.Message) *response {
	response := &response{
		session:     session,
		fromHandler: make(chan *asio.Message),
		toWorker:    toWorker,
		closed:      make(chan struct{}),
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
func (response *response) Write(data interface{}) {
	var res []byte
	codec.NewEncoderBytes(&res, h).Encode(&data)

	chunkMsg := &asio.Message{
		CommonMessageInfo: asio.CommonMessageInfo{
			Session: response.session,
			MsgType: ChunkType,
		},
		Payload: []interface{}{res},
	}

	// ToDo: check state of the stream
	// if it is closed there will be a deadlock
	response.fromHandler <- chunkMsg
}

// Notify a client about finishing the datastream.
func (response *response) Close() {
	chokeMsg := &asio.Message{
		CommonMessageInfo: asio.CommonMessageInfo{
			Session: response.session,
			MsgType: ChokeType,
		},
		Payload: []interface{}{},
	}

	// ToDo: check state of the stream
	// if it is closed there will be a deadlock
	response.fromHandler <- chokeMsg
	close(response.closed)
}

// Send error to a client. Specify code and message, which describes this error.
func (response *response) ErrorMsg(code int, msg string) {
	errorMessage := &asio.Message{
		CommonMessageInfo: asio.CommonMessageInfo{
			Session: response.session,
			MsgType: ErrorType,
		},
		Payload: []interface{}{code, msg},
	}

	// ToDo: check state of the stream
	// if it is closed there will be a deadlock
	response.fromHandler <- errorMessage
	close(response.closed)
}
