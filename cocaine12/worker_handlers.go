package cocaine12

import (
	"fmt"

	"golang.org/x/net/context"
)

// EventHandler represents a type of handler
type EventHandler func(context.Context, Request, Response)

// RequestHandler represents a type of handler
type RequestHandler func(context.Context, string, Request, Response)

// TerminationHandler invokes when termination message is received
type TerminationHandler func(context.Context)

type EventHandlers struct {
	fallback RequestHandler
	handlers map[string]EventHandler
}

func NewEventHandlersFromMap(handlers map[string]EventHandler) *EventHandlers {
	return &EventHandlers{DefaultFallbackHandler, handlers}
}

func NewEventHandlers() *EventHandlers {
	return NewEventHandlersFromMap(make(map[string]EventHandler))
}

func (e *EventHandlers) On(name string, handler EventHandler) {
	e.handlers[name] = handler
}

// SetFallbackHandler sets the handler to be a fallback handler
func (e *EventHandlers) SetFallbackHandler(handler RequestHandler) {
	e.fallback = handler
}

// DefaultFallbackHandler sends an error message if a client requests
// unhandled event
func DefaultFallbackHandler(ctx context.Context, event string, request Request, response Response) {
	errMsg := fmt.Sprintf("There is no handler for an event %s", event)
	response.ErrorMsg(ErrorNoEventHandler, errMsg)
}

func (e *EventHandlers) Call(ctx context.Context, event string, request Request, response Response) {
	handler := e.handlers[event]
	if handler == nil {
		e.fallback(ctx, event, request, response)
		return
	}
	handler(ctx, request, response)
}
