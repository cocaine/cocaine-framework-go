package cocaine12

import (
	"errors"
	"flag"
	"fmt"
	"time"
)

const (
	heartbeatTimeout      = time.Second * 20
	disownTimeout         = time.Second * 5
	coreConnectionTimeout = time.Second * 5

	// Error codes
	// no handler for requested event
	ErrorNoEventHandler = 200
	// panic in handler
	ErrorPanicInHandler = 100
)

var (
	ErrDisowned = errors.New("Disowned")
)

type RequestStream interface {
	push(*Message)
	Close()
}

type Request interface {
	Read(timeout ...time.Duration) ([]byte, error)
}

type ResponseStream interface {
	Write(data interface{})
	ErrorMsg(code int, message string)
	Close()
}

type Response ResponseStream

type EventHandler func(Request, Response)

// Performs IO operations between an application
// and cocaine-runtime, dispatches incoming messages
type Worker struct {
	// Connection to cocaine-runtime
	conn SocketIO
	// Id to introduce myself to cocaine-runtime
	id string
	// Each tick we shoud send a heartbeat as keep-alive
	heartbeatTimer *time.Timer
	// Timeout to receive a heartbeat reply
	disownTimer *time.Timer
	// Map handlers to sessions
	sessions map[uint64]RequestStream
	// Channel for replying from handlers
	// Everything is piped to cocaine without changes
	// ResponseStream is responsible to format proper message
	fromHandlers chan *Message
	// handlers
	handlers map[string]EventHandler
	// Notify Run about stop
	stopped chan struct{}
}

// Creates new Worker
func NewWorker() (*Worker, error) {
	setupFlags()
	flag.Parse()

	workerID := flagUUID

	// Connect to cocaine-runtime over a unix socket
	sock, err := NewUnixConnection(flagEndpoint, coreConnectionTimeout)
	if err != nil {
		return nil, err
	}
	return newWorker(sock, workerID)
}

func newWorker(conn SocketIO, id string) (*Worker, error) {
	w := &Worker{
		conn: conn,
		id:   id,

		heartbeatTimer: time.NewTimer(heartbeatTimeout),
		disownTimer:    time.NewTimer(disownTimeout),

		sessions:     make(map[uint64]RequestStream),
		fromHandlers: make(chan *Message),
		handlers:     make(map[string]EventHandler),

		stopped: make(chan struct{}),
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

// Bind handler for event
func (w *Worker) On(event string, handler EventHandler) {
	w.handlers[event] = handler
}

// Run serving loop
func (w *Worker) Run(handlers map[string]EventHandler) error {
	for event, handler := range handlers {
		w.On(event, handler)
	}

	return w.loop()
}

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
	w.onHeartbeat()

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
			w.onHeartbeat()

		case <-w.disownTimer.C:
			w.onDisown()
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
	case ChunkType:
		if reqStream, ok := w.sessions[msg.Session]; ok {
			reqStream.push(msg)
		}

	case ChokeType:
		if reqStream, ok := w.sessions[msg.Session]; ok {
			reqStream.Close()
			delete(w.sessions, msg.Session)
		}

	case InvokeType:
		var (
			event          string
			currentSession = msg.Session
		)

		event, ok := getEventName(msg)
		if !ok {
			// corrupted message
			return
		}

		responseStream := newResponse(currentSession, w.fromHandlers)

		handler, ok := w.handlers[event]
		if !ok {
			errMsg := fmt.Sprintf("There is no handler for event %s", event)
			responseStream.ErrorMsg(ErrorNoEventHandler, errMsg)
			return
		}

		requestStream := newRequest()
		w.sessions[currentSession] = requestStream

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errMsg := fmt.Sprintf("Error in event: '%s', exception: %s", event, r)
					responseStream.ErrorMsg(ErrorPanicInHandler, errMsg)
				}
			}()

			handler(requestStream, responseStream)
		}()

	case HeartbeatType:
		// Reply to heartbeat has been received,
		// so we are not disowned & disownTimer must be stopped
		// It will be launched when a next heartbeat is sent
		w.disownTimer.Stop()

	case TerminateType:
		// According to spec we have time
		// to prepare for being killed by cocaine-runtime
		w.onTerminate()

	default:
		// Invalid message
	}
}

// A reply to heartbeat is not arrived during disownTimeout,
// so it seems cocaine-runtime has died
func (w *Worker) onDisown() {
	w.Stop()
}

func (w *Worker) onTerminate() {
	w.Stop()
}

// Send handshake message to cocaine-runtime
// It is needed to be called only once on a startup
// to notify runtime that we have started
func (w *Worker) sendHandshake() {
	select {
	case w.conn.Write() <- NewHandshake(w.id):
	case <-w.conn.IsClosed():
	}
}

func (w *Worker) onHeartbeat() {
	select {
	case w.conn.Write() <- NewHeartbeatMessage():
	case <-w.conn.IsClosed():
	}

	// Wait for the reply until disown timeout comes
	w.disownTimer.Reset(disownTimeout)
	// Send next heartbeat over heartbeatTimeout
	w.heartbeatTimer.Reset(heartbeatTimeout)
}
