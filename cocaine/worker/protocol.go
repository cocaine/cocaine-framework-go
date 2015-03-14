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
