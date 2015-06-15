package cocaine12

import (
	"fmt"
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

type v0Protocol struct{}

func newV0Protocol() protocolDispather {
	return &v0Protocol{}
}

func (v *v0Protocol) onMessage(p protocolHandler, msg *Message) {
	switch msg.MsgType {
	case chunkType:
		p.onChunk(msg)

	case chokeType:
		p.onChoke(msg)

	case invokeType:
		p.onInvoke(msg)

	case heartbeatType:
		p.onHeartbeat(msg)

	case terminateType:
		p.onTerminate(msg)

	default:
		// Invalid message
		fmt.Printf("invalid message type: %d, message %v", msg.MsgType, msg)
	}
}

func (v *v0Protocol) isChunk(msg *Message) bool {
	return msg.MsgType == chunkType
}

func (v *v0Protocol) newHandshake(id string) *Message {
	return newHandshakeV0(id)
}

func (v *v0Protocol) newHeartbeat() *Message {
	return newHeartbeatV0()
}

func (v *v0Protocol) newChoke(session uint64) *Message {
	return newChokeV0(session)
}

func (v *v0Protocol) newChunk(session uint64, data []byte) *Message {
	return newChunkV0(session, data)
}

func (v *v0Protocol) newError(session uint64, category, code int, message string) *Message {
	return newErrorV0(session, code, message)
}

func newHandshakeV0(id string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: 0,
			MsgType: handshakeType,
		},
		Payload: []interface{}{id},
	}
}

func newHeartbeatV0() *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: 0,
			MsgType: heartbeatType,
		},
		Payload: []interface{}{},
	}
}

func newInvokeV0(session uint64, event string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: invokeType,
		},
		Payload: []interface{}{event},
	}
}

func newChunkV0(session uint64, data []byte) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: chunkType,
		},
		Payload: []interface{}{data},
	}
}

func newErrorV0(session uint64, code int, message string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: errorType,
		},
		Payload: []interface{}{code, message},
	}
}

func newChokeV0(session uint64) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: chokeType,
		},
		Payload: []interface{}{},
	}
}
