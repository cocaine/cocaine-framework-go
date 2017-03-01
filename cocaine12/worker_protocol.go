package cocaine12

import (
	"fmt"

	"github.com/tinylib/msgp/msgp"
)

const (
	v0 = 0
	v1 = 1
)

type ErrRequest struct {
	Message        string
	Category, Code uint64
}

var _ msgp.Unmarshaler = &ErrRequest{}

// UnmarshalMsg implements msgp.Unmarshaler
func (e *ErrRequest) UnmarshalMsg(b []byte) ([]byte, error) {
	sz, b, err := msgp.ReadArrayHeaderBytes(b)
	if err != nil || sz != 2 {
		return b, ErrMalformedErrorMessage
	}

	// read category and code
	sz, b, err = msgp.ReadArrayHeaderBytes(b)
	if err != nil || sz != 2 {
		return b, ErrMalformedErrorMessage
	}

	// category
	var n msgp.Number
	b, err = n.UnmarshalMsg(b)
	if err != nil {
		return b, ErrMalformedErrorMessage
	}
	e.Category, _ = n.Uint()

	// code
	b, err = n.UnmarshalMsg(b)
	if err != nil {
		return b, ErrMalformedErrorMessage
	}
	e.Code, _ = n.Uint()
	// read message
	e.Message, b, err = msgp.ReadStringBytes(b)
	return b, err
}

func (e *ErrRequest) Error() string {
	return fmt.Sprintf("[%d] [%d] %s", e.Category, e.Code, e.Message)
}

type eventName string

var _ msgp.Unmarshaler = new(eventName)

func (e *eventName) UnmarshalMsg(b []byte) ([]byte, error) {
	sz, b, err := msgp.ReadArrayHeaderBytes(b)
	if err != nil || sz != 1 {
		return b, fmt.Errorf("mailformed invoke message %v %d", err, sz)
	}

	name, b, err := msgp.ReadStringBytes(b)
	if err != nil {
		return b, fmt.Errorf("mailformed event name in invoke message %v", err)
	}

	(*e) = eventName(name)
	return b, nil
}

type messageTypeDetector interface {
	isChunk(msg *message) bool
}

type protocolHandler interface {
	onChoke(msg *message)
	onChunk(msg *message)
	onError(msg *message)
	onHeartbeat(msg *message)
	onInvoke(msg *message) error
	onTerminate(msg *message)
}

type utilityProtocolGenerator interface {
	newHandshake(id string) *message
	newHeartbeat() *message
}

type handlerProtocolGenerator interface {
	messageTypeDetector
	newChoke(session uint64) *message
	newChunk(session uint64, data []byte) *message
	newError(session uint64, category, code int, message string) *message
}

type protocolDispather interface {
	utilityProtocolGenerator
	handlerProtocolGenerator
	onMessage(p protocolHandler, msg *message) error
}
