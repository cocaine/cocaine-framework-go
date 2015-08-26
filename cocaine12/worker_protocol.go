package cocaine12

import (
	"fmt"
)

const (
	v0 = 0
	v1 = 1
)

type ErrRequest struct {
	Message        string
	Category, Code int
}

func (e *ErrRequest) Error() string {
	return fmt.Sprintf("[%d] [%d] %s", e.Category, e.Code, e.Message)
}

type messageTypeDetector interface {
	isChunk(msg *Message) bool
}

type protocolHandler interface {
	onChoke(msg *Message)
	onChunk(msg *Message)
	onError(msg *Message)
	onHeartbeat(msg *Message)
	onInvoke(msg *Message) error
	onTerminate(msg *Message)
}

type utilityProtocolGenerator interface {
	newHandshake(id string) *Message
	newHeartbeat() *Message
}

type handlerProtocolGenerator interface {
	messageTypeDetector
	newChoke(session uint64) *Message
	newChunk(session uint64, data []byte) *Message
	newError(session uint64, category, code int, message string) *Message
}

type protocolDispather interface {
	utilityProtocolGenerator
	handlerProtocolGenerator
	onMessage(p protocolHandler, msg *Message) error
}

func getEventName(msg *Message) (string, bool) {
	switch event := msg.Payload[0].(type) {
	case string:
		return event, true
	case []uint8:
		return string(event), true
	}
	return "", false
}
