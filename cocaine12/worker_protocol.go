package cocaine

import (
	"github.com/cocaine/cocaine-framework-go/vendor/src/github.com/ugorji/go/codec"
)

var (
	mh codec.MsgpackHandle
	h  = &mh
)

const (
	HandshakeType = iota
	HeartbeatType
	TerminateType
	InvokeType
	ChunkType
	ErrorType
	ChokeType
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

func NewHandshake(id string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: 0,
			MsgType: HandshakeType,
		},
		Payload: []interface{}{id},
	}
}

func NewHeartbeatMessage() *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: 0,
			MsgType: HeartbeatType,
		},
		Payload: []interface{}{},
	}
}

func NewInvoke(session uint64, event string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: InvokeType,
		},
		Payload: []interface{}{event},
	}
}

func NewChunk(session uint64, data interface{}) *Message {
	var res []byte
	codec.NewEncoderBytes(&res, h).Encode(data)

	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: ChunkType,
		},
		Payload: []interface{}{res},
	}
}

func NewError(session uint64, code int, message string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: ErrorType,
		},
		Payload: []interface{}{code, message},
	}
}

func NewChoke(session uint64) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: ChokeType,
		},
		Payload: []interface{}{},
	}
}
