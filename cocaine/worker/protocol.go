package worker

import (
	"github.com/cocaine/cocaine-framework-go/cocaine/asio"
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

func getEventName(msg *asio.Message) (string, bool) {
	event, ok := msg.Payload[0].(string)
	return event, ok
}

// ToDo: find out if sync.Pool may give
// profit to create messages

func NewHandshake(id string) *asio.Message {
	return &asio.Message{
		CommonMessageInfo: asio.CommonMessageInfo{
			Session: 0,
			MsgType: HandshakeType,
		},
		Payload: []interface{}{id},
	}
}

func NewHeartbeatMessage() *asio.Message {
	return &asio.Message{
		CommonMessageInfo: asio.CommonMessageInfo{
			Session: 0,
			MsgType: HeartbeatType,
		},
		Payload: []interface{}{},
	}
}

func NewInvoke(session uint64, event string) *asio.Message {
	return &asio.Message{
		CommonMessageInfo: asio.CommonMessageInfo{
			Session: session,
			MsgType: InvokeType,
		},
		Payload: []interface{}{event},
	}
}

func NewChunk(session uint64, payload []interface{}) *asio.Message {
	return &asio.Message{
		CommonMessageInfo: asio.CommonMessageInfo{
			Session: session,
			MsgType: ChunkType,
		},
		Payload: payload,
	}
}

func NewError(session uint64, code int, message string) *asio.Message {
	return &asio.Message{
		CommonMessageInfo: asio.CommonMessageInfo{
			Session: session,
			MsgType: ErrorType,
		},
		Payload: []interface{}{code, message},
	}
}

func NewChoke(session uint64) *asio.Message {
	return &asio.Message{
		CommonMessageInfo: asio.CommonMessageInfo{
			Session: session,
			MsgType: ChokeType,
		},
		Payload: []interface{}{},
	}
}
