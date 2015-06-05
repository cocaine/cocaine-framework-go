package cocaine12

import (
	"github.com/cocaine/cocaine-framework-go/vendor/src/github.com/ugorji/go/codec"
)

var (
	mh codec.MsgpackHandle
	h  = &mh
)

const (
	handshakeType = iota
	heartbeatType
	terminateType
	invokeType
	chunkType
	errorType
	chokeType
)

func getEventName(msg *Message) (string, bool) {
	switch event := msg.Payload[0].(type) {
	case string:
		return event, true
	case []uint8:
		return string(event), true
	}
	return "", false
}

// ToDo: find out if sync.Pool may give
// profit to create messages

func newHandshake(id string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: 0,
			MsgType: handshakeType,
		},
		Payload: []interface{}{id},
	}
}

func newHeartbeatMessage() *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: 0,
			MsgType: heartbeatType,
		},
		Payload: []interface{}{},
	}
}

func newInvoke(session uint64, event string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: invokeType,
		},
		Payload: []interface{}{event},
	}
}

func newChunk(session uint64, data []byte) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: chunkType,
		},
		Payload: []interface{}{data},
	}
}

func newError(session uint64, code int, message string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: errorType,
		},
		Payload: []interface{}{code, message},
	}
}

func newChoke(session uint64) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: chokeType,
		},
		Payload: []interface{}{},
	}
}
