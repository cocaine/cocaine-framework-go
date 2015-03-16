package cocaine

func loop(input <-chan *Message, output chan *Message, onclose <-chan struct{}) {
	var (
		pending []*Message
		closed  <-chan struct{} = onclose
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

type request struct {
	fromWorker chan *Message
	toHandler  chan *Message
	closed     chan struct{}
}

func newRequest() *request {
	request := &request{
		fromWorker: make(chan *Message),
		toHandler:  make(chan *Message),
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

func (request *request) Read() chan *Message {
	return request.toHandler
}

func (request *request) Push(msg *Message) {
	request.fromWorker <- msg
}

func (request *request) Close() {
	close(request.closed)
}

type response struct {
	session     uint64
	fromHandler chan *Message
	toWorker    chan *Message
	closed      chan struct{}
}

func newResponse(session uint64, toWorker chan *Message) *response {
	response := &response{
		session:     session,
		fromHandler: make(chan *Message),
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
func (r *response) Write(data interface{}) {
	if r.isClosed() {
		return
	}

	r.fromHandler <- NewChunk(r.session, data)
}

// Notify a client about finishing the datastream.
func (r *response) Close() {
	if r.isClosed() {
		return
	}

	r.fromHandler <- NewChoke(r.session)
	close(r.closed)
}

// Send error to a client. Specify code and message, which describes this error.
func (r *response) ErrorMsg(code int, message string) {
	if r.isClosed() {
		return
	}

	r.fromHandler <- NewError(
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
