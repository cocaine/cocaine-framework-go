package cocaine12

import (
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"golang.org/x/net/context"
)

const (
	heartbeatTimeout      = time.Second * 10
	disownTimeout         = time.Second * 5
	coreConnectionTimeout = time.Second * 5

	// ErrorNoEventHandler returns when there is no handler for a given event
	ErrorNoEventHandler = 200
	// ErrorPanicInHandler returns when a handler is recovered from panic
	ErrorPanicInHandler = 100
)

var (
	// ErrDisowned raises when the worker doesn't receive
	// a heartbeat message during a heartbeat timeout
	ErrDisowned = errors.New("disowned")
)

type requestStream interface {
	push(*Message)
	Close()
}

// Request provides an interface for a handler to get data
type Request interface {
	Read(timeout ...time.Duration) ([]byte, error)
}

// ResponseStream provides an interface for a handler to reply
type ResponseStream interface {
	Write(data []byte)
	ErrorMsg(code int, message string)
	Close()
}

// Response provides an interface for a handler to reply
type Response ResponseStream

// EventHandler represents a type of handler
type EventHandler func(context.Context, Request, Response)

// FallbackEventHandler handles an event if there is no other handler
// for the given event
type FallbackEventHandler func(context.Context, string, Request, Response)

// DefaultFallbackEventHandler sends an error message if a client requests
// unhandled event
func DefaultFallbackEventHandler(ctx context.Context, event string, request Request, response Response) {
	errMsg := fmt.Sprintf("There is no handler for an event %s", event)
	response.ErrorMsg(ErrorNoEventHandler, errMsg)
}

func recoverTrap(ctx context.Context, event string, response Response, printStack bool) {
	if recoverInfo := recover(); recoverInfo != nil {
		var stack []byte

		if printStack {
			stack = debug.Stack()
		}

		response.ErrorMsg(
			ErrorPanicInHandler,
			fmt.Sprintf("Event: '%s', recover: %s, stack: %s", event, recoverInfo, stack),
		)
	}
}

// Worker performs IO operations between an application
// and cocaine-runtime, dispatches incoming messages
type Worker struct {
	// Connection to cocaine-runtime
	conn socketIO
	// Id to introduce myself to cocaine-runtime
	id string
	// Each tick we shoud send a heartbeat as keep-alive
	heartbeatTimer *time.Timer
	// Timeout to receive a heartbeat reply
	disownTimer *time.Timer
	// Map handlers to sessions
	sessions map[uint64]requestStream
	// Channel for replying from handlers
	// Everything is piped to cocaine without changes
	// ResponseStream is responsible to format proper message
	fromHandlers chan *Message
	// handlers
	handlers map[string]EventHandler
	// Notify Run about stop
	stopped chan struct{}
	// FallbackEventHandler handles an event if there is no other handler
	fallbackHandler FallbackEventHandler
	// if set recoverTrap sends Stack
	debug bool
}

// NewWorker connects to the cocaine-runtime and create Worker on top of this connection
func NewWorker() (*Worker, error) {
	workerID := defaults.UUID

	unixSocketEndpoint := defaults.Endpoint
	if unixSocketEndpoint == "" {
		return nil, fmt.Errorf("cocaine endpoint must be specified")
	}

	// Connect to cocaine-runtime over a unix socket
	sock, err := newUnixConnection(unixSocketEndpoint, coreConnectionTimeout)
	if err != nil {
		return nil, err
	}

	return newWorker(sock, workerID, defaults.Protocol)
}

func newWorker(conn socketIO, id string, version int) (*Worker, error) {
	w := &Worker{
		conn: conn,
		id:   id,

		heartbeatTimer: time.NewTimer(heartbeatTimeout),
		disownTimer:    time.NewTimer(disownTimeout),

		sessions:     make(map[uint64]requestStream),
		fromHandlers: make(chan *Message),
		handlers:     make(map[string]EventHandler),

		stopped: make(chan struct{}),

		fallbackHandler: DefaultFallbackEventHandler,
		debug:           false,
	}

	// NewTimer launches timer
	// but it should be started after
	// we send heartbeat message
	w.disownTimer.Stop()
	// It will be reset in onHeartbeat()
	// after worker runs
	w.heartbeatTimer.Stop()

	// Send handshake to notify cocaine-runtime
	// that we have started
	w.sendHandshake()

	return w, nil
}

// On binds the handler for a given event
func (w *Worker) On(event string, handler EventHandler) {
	w.handlers[event] = handler
}

// SetFallbackHandler sets the handler to be a fallback handler
func (w *Worker) SetFallbackHandler(handler FallbackEventHandler) {
	w.fallbackHandler = handler
}

// call a fallback handler inwith a panic trap
func (w *Worker) callFallbackHandler(ctx context.Context, event string, request Request, response Response) {
	defer response.Close()
	defer recoverTrap(ctx, event, response, w.debug)
	w.fallbackHandler(ctx, event, request, response)
}

// SetDebug enables debug mode of the Worker.
// It allows to print Stack of a paniced handler
func (w *Worker) SetDebug(debug bool) {
	w.debug = debug
}

// Run makes the worker anounce itself to a cocaine-runtime
// as being ready to hadnle incoming requests and hablde them
func (w *Worker) Run(handlers map[string]EventHandler) error {
	for event, handler := range handlers {
		w.On(event, handler)
	}

	return w.loop()
}

// Stop makes the Worker stop handling requests
func (w *Worker) Stop() {
	if w.isStopped() {
		return
	}

	close(w.stopped)
	w.conn.Close()
}

func (w *Worker) isStopped() bool {
	select {
	case <-w.stopped:
		return true
	default:
	}
	return false
}

func (w *Worker) loop() error {
	var err error
	// Send heartbeat to notify cocaine-runtime
	// we are ready to work
	w.onHeartbeatTimeout()

	for {
		select {
		case msg, ok := <-w.conn.Read():
			if ok {
				// otherwise the connection is closed
				w.onMessage(msg)
			}

		case <-w.heartbeatTimer.C:
			// Reset (start) disown & heartbeat timers
			// Send a heartbeat message to cocaine-runtime
			w.onHeartbeatTimeout()

		case <-w.disownTimer.C:
			w.onDisownTimeout()
			err = ErrDisowned

		// ToDo: reply directly to a connection
		case outcoming := <-w.fromHandlers:
			select {
			case w.conn.Write() <- outcoming:
			// Socket is in closed state, so drop data
			case <-w.conn.IsClosed():
			}
		case <-w.stopped:
			// If worker is disowned
			// err is set to ErrDisowned
			return err
		}
	}
}

func (w *Worker) onMessage(msg *Message) {
	switch msg.MsgType {
	case chunkType:
		w.onChunk(msg)

	case chokeType:
		w.onChoke(msg)

	case invokeType:
		w.onInvoke(msg)

	case heartbeatType:
		w.onHeartbeat(msg)

	case terminateType:
		w.onTerminate(msg)

	default:
		// Invalid message
		fmt.Printf("invalid message type: %d, message %v", msg.MsgType, msg)
	}
}

// A reply to heartbeat is not arrived during disownTimeout,
// so it seems cocaine-runtime has died
func (w *Worker) onDisownTimeout() {
	w.Stop()
}

func (w *Worker) onHeartbeatTimeout() {
	select {
	case w.conn.Write() <- newHeartbeatMessage():
	case <-w.conn.IsClosed():
	}

	// Wait for the reply until disown timeout comes
	w.disownTimer.Reset(disownTimeout)
	// Send next heartbeat over heartbeatTimeout
	w.heartbeatTimer.Reset(heartbeatTimeout)
}

// Send handshake message to cocaine-runtime
// It is needed to be called only once on a startup
// to notify runtime that we have started
func (w *Worker) sendHandshake() {
	select {
	case w.conn.Write() <- newHandshake(w.id):
	case <-w.conn.IsClosed():
	}
}

// Message handlers

func (w *Worker) onChunk(msg *Message) {
	if reqStream, ok := w.sessions[msg.Session]; ok {
		reqStream.push(msg)
	}
}

func (w *Worker) onChoke(msg *Message) {
	if reqStream, ok := w.sessions[msg.Session]; ok {
		reqStream.Close()
		delete(w.sessions, msg.Session)
	}
}

func (w *Worker) onInvoke(msg *Message) {
	var (
		event          string
		currentSession = msg.Session
	)

	event, ok := getEventName(msg)
	if !ok {
		// corrupted message
		return
	}

	ctx := context.Background()
	responseStream := newResponse(currentSession, w.fromHandlers)
	requestStream := newRequest()
	w.sessions[currentSession] = requestStream

	handler, ok := w.handlers[event]
	if !ok {
		go w.callFallbackHandler(ctx, event, requestStream, responseStream)
		return
	}

	go func() {
		defer responseStream.Close()
		defer recoverTrap(ctx, event, responseStream, w.debug)

		handler(ctx, requestStream, responseStream)
	}()
}

func (w *Worker) onHeartbeat(msg *Message) {
	// Reply to a heartbeat has been received,
	// so we are not disowned & disownTimer must be stopped
	// It will be launched when the next heartbeat is sent
	w.disownTimer.Stop()
}

func (w *Worker) onTerminate(msg *Message) {
	// According to spec we have time
	// to prepare for being killed by cocaine-runtime
	w.Stop()
}
