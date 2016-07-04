package cocaine12

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/net/context"
)

const (
	heartbeatTimeout      = time.Second * 10
	disownTimeout         = time.Second * 5
	coreConnectionTimeout = time.Second * 5
	terminationTimeout    = time.Second * 5

	// ErrorNoEventHandler returns when there is no handler for a given event
	ErrorNoEventHandler = 200
	// ErrorPanicInHandler returns when a handler is recovered from panic
	ErrorPanicInHandler = 100
)

var (
	// ErrDisowned raises when the worker doesn't receive
	// a heartbeat message during a heartbeat timeout
	ErrDisowned = errors.New("disowned from cocaine-runtime")
	// ErrNoCocaineEndpoint means that the worker doesn't know an endpoint
	// to Cocaine
	ErrNoCocaineEndpoint = errors.New("cocaine endpoint must be specified")
	// ErrConnectionLost means that the connection between the worker and
	// runtime has been lost
	ErrConnectionLost = errors.New("the connection to runtime has been lost")
)

type requestStream interface {
	push(*Message)
	Close()
}

// Request provides an interface for a handler to get data
type Request interface {
	Read(ctx context.Context) ([]byte, error)
}

// ResponseStream provides an interface for a handler to reply
type ResponseStream interface {
	io.WriteCloser
	// ZeroCopyWrite sends data to a client.
	// Response takes the ownership of the buffer, so provided buffer must not be edited.
	ZeroCopyWrite(data []byte) error
	ErrorMsg(code int, message string) error
}

// Response provides an interface for a handler to reply
type Response ResponseStream

func trapRecoverAndClose(ctx context.Context, event string, response Response, printStack bool) {
	if recoverInfo := recover(); recoverInfo != nil {
		var stack []byte

		if printStack {
			stack = make([]byte, 4096)
			stackSize := runtime.Stack(stack, false)
			stack = stack[:stackSize]
		}

		response.ErrorMsg(
			ErrorPanicInHandler,
			fmt.Sprintf("Event: '%s', recover: %s, stack: \n%s\n", event, recoverInfo, stack),
		)
		return
	}

	response.Close()
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
	// handler
	handler RequestHandler
	// Notify Run about stop
	stopped chan struct{}
	// if set recoverTrap sends Stack
	debug bool
	// allow the worker to handle SIGUSR1 to print all goroutines stacks
	stackSignalEnabled bool
	// protocol version id
	protoVersion int
	// protocol dispatcher
	dispatcher protocolDispather
	// temination handler
	terminationHandler TerminationHandler
}

// NewWorker connects to the cocaine-runtime and create Worker on top of this connection
func NewWorker() (*Worker, error) {
	workerID := GetDefaults().UUID()

	unixSocketEndpoint := GetDefaults().Endpoint()
	if unixSocketEndpoint == "" {
		return nil, ErrNoCocaineEndpoint
	}

	// Connect to cocaine-runtime over a unix socket
	sock, err := newUnixConnection(unixSocketEndpoint, coreConnectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to Cocaine via %s: %v",
			unixSocketEndpoint, err)
	}

	return newWorker(sock, workerID,
		GetDefaults().Protocol(),
		GetDefaults().Debug())
}

func newWorker(conn socketIO, id string, protoVersion int, debug bool) (*Worker, error) {
	w := &Worker{
		conn: conn,
		id:   id,

		heartbeatTimer: time.NewTimer(heartbeatTimeout),
		disownTimer:    time.NewTimer(disownTimeout),

		sessions: make(map[uint64]requestStream),

		stopped: make(chan struct{}),

		debug:              debug,
		stackSignalEnabled: true,

		protoVersion:       protoVersion,
		dispatcher:         nil,
		terminationHandler: nil,
	}

	switch w.protoVersion {
	case v1:
		w.dispatcher = newV1Protocol()
	default:
		return nil, fmt.Errorf("unsupported protocol version %d", w.protoVersion)
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
	if err := w.sendHandshake(); err != nil {
		return nil, err
	}

	return w, nil
}

// SetDebug enables debug mode of the Worker.
// It allows to print Stack of a paniced handler
func (w *Worker) SetDebug(debug bool) {
	w.debug = debug
}

// EnableStackSignal allows/disallows the worker to catch
// SIGUSR1 to print all goroutines stacks. It's enabled by default.
// This function must be called before Worker.Run to take effect.
func (w *Worker) EnableStackSignal(enable bool) {
	w.stackSignalEnabled = enable
}

// Run makes the worker anounce itself to a cocaine-runtime
// as being ready to hadnle incoming requests and hablde them
// terminationHandler allows to attach handler which will be called
// when SIGTERM arrives
func (w *Worker) Run(handler RequestHandler, terminationHandler TerminationHandler) error {
	w.handler = handler
	w.terminationHandler = terminationHandler
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
	// Send heartbeat to notify cocaine-runtime
	// we are ready to work
	w.onHeartbeatTimeout()

	var stackSignal chan os.Signal

	if w.stackSignalEnabled {
		stackSignal = make(chan os.Signal, 1)
		signal.Notify(stackSignal, syscall.SIGUSR1)
		defer signal.Stop(stackSignal)
	}

	for {
		select {
		case msg, ok := <-w.conn.Read():
			if !ok {
				// either the connection is lost
				// or the worker was stopped
				select {
				case <-w.stopped:
					return nil
				default:
					return ErrConnectionLost
				}
			}

			// non-blocking
			if err := w.dispatcher.onMessage(w, msg); err != nil {
				fmt.Printf("onMessage returns %v\n", err)
			}

		case <-w.heartbeatTimer.C:
			// Reset (start) disown & heartbeat timers
			// Send a heartbeat message to cocaine-runtime
			w.onHeartbeatTimeout() // non-blocking

		case <-w.disownTimer.C:
			w.onDisownTimeout() // non-blocking
			return ErrDisowned

		case <-w.stopped:
			return nil

		case <-stackSignal:
			w.printAllStacks()
		}
	}
}

// printAllStacks prints all stacks to stderr and writes to a file
func (w *Worker) printAllStacks() {
	stackTrace := dumpStack()
	// print to stdout to have it in the logs
	fmt.Printf("=== START STACKTRACE ===\n%s\n=== END STACKTRACE ===", stackTrace)
	// to debug blocked workers. It will be removed somewhen
	filename := fmt.Sprintf("%s-%d", GetDefaults().ApplicationName(), os.Getpid())
	if err := ioutil.WriteFile(filename, stackTrace, 0660); err != nil {
		fmt.Printf("unable to create the file with stacktraces %s: %v\n", filename, err)
	}
}

// A reply to heartbeat is not arrived during disownTimeout,
// so it seems cocaine-runtime has died
func (w *Worker) onDisownTimeout() {
	w.Stop()
}

func (w *Worker) onHeartbeatTimeout() {
	// Wait for the reply until disown timeout comes
	w.disownTimer.Reset(disownTimeout)
	// Send next heartbeat over heartbeatTimeout
	w.heartbeatTimer.Reset(heartbeatTimeout)

	select {
	case w.conn.Write() <- w.dispatcher.newHeartbeat():
	case <-w.conn.IsClosed():
	case <-time.After(disownTimeout):
	}
}

// Send handshake message to cocaine-runtime
// It is needed to be called only once on a startup
// to notify runtime that we have started
func (w *Worker) sendHandshake() error {
	select {
	case w.conn.Write() <- w.dispatcher.newHandshake(w.id):
	case <-w.conn.IsClosed():
	case <-time.After(disownTimeout):
		return fmt.Errorf("unable to send a handshake for a long time")
	}
	return nil
}

// Message handlers

func (w *Worker) onChoke(msg *Message) {
	if reqStream, ok := w.sessions[msg.Session]; ok {
		reqStream.Close()
		delete(w.sessions, msg.Session)
	}
}

func (w *Worker) onChunk(msg *Message) {
	if reqStream, ok := w.sessions[msg.Session]; ok {
		reqStream.push(msg)
	}
}

func (w *Worker) onError(msg *Message) {
	if reqStream, ok := w.sessions[msg.Session]; ok {
		reqStream.push(msg)
	}
}

func (w *Worker) onInvoke(msg *Message) error {
	event, ok := getEventName(msg)
	if !ok {
		// corrupted message
		return fmt.Errorf("unable to get an event name from %s", msg.String())
	}

	var (
		currentSession = msg.Session
		ctx            context.Context
	)

	ctx = context.Background()

	if traceInfo, err := msg.Headers.getTraceData(); err == nil {
		ctx = AttachTraceInfo(ctx, traceInfo)
	}

	responseStream := newResponse(w.dispatcher, currentSession, w.conn)
	requestStream := newRequest(w.dispatcher)
	w.sessions[currentSession] = requestStream

	go func() {
		// this trap catches a panic from a handler
		// and checks if the response is closed.
		defer trapRecoverAndClose(ctx, event, responseStream, w.debug)

		ctx, closeHandlerSpan := NewSpan(ctx, event)
		defer closeHandlerSpan()

		w.handler(ctx, event, requestStream, responseStream)
	}()
	return nil
}

func (w *Worker) onHeartbeat(msg *Message) {
	// Reply to a heartbeat has been received,
	// so we are not disowned & disownTimer must be stopped
	// It will be launched when the next heartbeat is sent
	w.disownTimer.Stop()
}

func (w *Worker) onTerminate(msg *Message) {
	if w.terminationHandler != nil {
		ctx, cancelTimeout := context.WithTimeout(context.Background(), terminationTimeout)
		onDone := make(chan struct{})
		go func() {
			w.terminationHandler(ctx)
			close(onDone)
		}()

		select {
		case <-onDone:
			cancelTimeout()
		case <-ctx.Done():
			fmt.Printf("terminationHandler timeouted: %v\n", ctx.Err())
		}
	}

	// According to spec we have time
	// to prepare for being killed by cocaine-runtime
	select {
	case w.conn.Write() <- msg:
		// reply with the same termination message
	case <-w.conn.IsClosed():
	case <-time.After(disownTimeout):
	}
	w.Stop()
}
