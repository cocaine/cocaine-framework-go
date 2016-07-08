package cocaine12

// Worker performs IO operations between an application
// and cocaine-runtime, dispatches incoming messages
// This is an adapter to WorkerNG
type Worker struct {
	impl               *WorkerNG
	handlers           *EventHandlers
	terminationHandler TerminationHandler
}

// NewWorker connects to the cocaine-runtime and create WorkerNG on top of this connection
func NewWorker() (*Worker, error) {
	impl, err := NewWorkerNG()
	if err != nil {
		return nil, err
	}
	return &Worker{impl, NewEventHandlers(), nil}, nil
}

// Used in tests only
func newWorker(conn socketIO, id string, protoVersion int, debug bool) (*Worker, error) {
	impl, err := newWorkerNG(conn, id, protoVersion, debug)
	if err != nil {
		return nil, err
	}
	return &Worker{impl, NewEventHandlers(), nil}, nil
}

// SetDebug enables debug mode of the Worker.
// It allows to print Stack of a paniced handler
func (w *Worker) SetDebug(debug bool) {
	w.impl.SetDebug(debug)
}

// EnableStackSignal allows/disallows the worker to catch
// SIGUSR1 to print all goroutines stacks. It's enabled by default.
// This function must be called before Worker.Run to take effect.
func (w *Worker) EnableStackSignal(enable bool) {
	w.impl.EnableStackSignal(enable)
}

// SetTerminationHandler allows to attach handler which will be called
// when SIGTERM arrives
func (w *Worker) SetTerminationHandler(handler TerminationHandler) {
	w.terminationHandler = handler
}

// On binds the handler for a given event
func (w *Worker) On(event string, handler EventHandler) {
	w.handlers.On(event, handler)
}

// SetFallbackHandler sets the handler to be a fallback handler
func (w *Worker) SetFallbackHandler(handler FallbackEventHandler) {
	w.handlers.SetFallbackHandler(RequestHandler(handler))
}

func (w *Worker) Run(handlers map[string]EventHandler) error {
	for event, handler := range handlers {
		w.On(event, handler)
	}
	return w.impl.Run(w.handlers.Call, w.terminationHandler)
}

// Stop makes the Worker stop handling requests
func (w *Worker) Stop() {
	w.impl.Stop()
}
