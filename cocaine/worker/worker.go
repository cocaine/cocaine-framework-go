package worker

import (
	"flag"
	"fmt"
	"time"

	"github.com/cocaine/cocaine-framework-go/cocaine/asio"
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

type RequestStream interface {
	Push(*asio.Message)
	Close()
}

type ResponseStream interface {
	Write(data interface{})
	ErrorMsg(code int, message string)
	Close()
}

type EventHandler func(RequestStream, ResponseStream)

// Performs IO operations between an application
// and cocaine-runtime, dispatches incoming messages
type Worker struct {
	// Connection to cocaine-runtime
	asio.SocketIO
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
	fromHandlers chan *asio.Message
	// handlers
	handlers map[string]EventHandler
	// Notify Run about stop
	stopped chan struct{}
}

// Creates new Worker
func NewWorker() (*Worker, error) {
	flag.Parse()

	workerID := flagUUID

	// Connect to cocaine-runtime over a unix socket
	sock, err := asio.NewUnixConnection(flagEndpoint, coreConnectionTimeout)
	if err != nil {
		return nil, err
	}

	w := &Worker{
		SocketIO: sock,
		id:       workerID,

		heartbeatTimer: time.NewTimer(heartbeatTimeout),
		disownTimer:    time.NewTimer(disownTimeout),

		sessions:     make(map[uint64]RequestStream),
		fromHandlers: make(chan *asio.Message),
		handlers:     make(map[string]EventHandler),

		stopped: make(chan struct{}),
	}

	// NewTimer launches timer
	// but it should be started after
	// we send heartbeat message
	w.disownTimer.Stop()

	// Send handshake to notify cocaine-runtime
	// that we have started
	w.sendHandshake()

	// Send heartbeat to notify cocaine-runtime
	// we are ready to work
	w.onHeartbeat()
	return w, nil
}

func (w *Worker) Run(handlers map[string]EventHandler) {

	for event, handler := range handlers {
		w.handlers[event] = handler
	}

	for {
		select {
		case msg := <-w.Read():
			w.onMessage(msg)

		case <-w.heartbeatTimer.C:
			// Reset (start) disown & heartbeat timers
			// Send a heartbeat message to cocaine-runtime
			w.onHeartbeat()

		case <-w.disownTimer.C:
			w.onDisown()

		// ToDo: reply directly to a connection
		case outcoming := <-w.fromHandlers:
			w.Write() <- outcoming

		case <-w.stopped:
			return
		}
	}
}

func (w *Worker) Stop() {
	close(w.stopped)
}

func (w *Worker) onMessage(msg *asio.Message) {
	switch msg.MsgType {
	case ChunkType:
		if reqStream, ok := w.sessions[msg.Session]; ok {
			reqStream.Push(msg)
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
	w.Close()
}

func (w *Worker) onTerminate() {
	w.Close()
}

// Send handshake message to cocaine-runtime
// It is needed to be called only once on a startup
// to notify runtime that we have started
func (w *Worker) sendHandshake() {
	w.Write() <- NewHandshake(w.id)
}

func (w *Worker) onHeartbeat() {
	w.Write() <- NewHeartbeatMessage()

	// Wait for the reply until disown timeout comes
	w.disownTimer.Reset(disownTimeout)
	// Send next heartbeat over heartbeatTimeout
	w.heartbeatTimer.Reset(heartbeatTimeout)
}
