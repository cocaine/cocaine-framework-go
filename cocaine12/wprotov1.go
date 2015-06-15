package cocaine12

import (
	"fmt"
)

const (
	v1Handshake = 0
	v1Heartbeat = 0
	v1Invoke    = 0
	v1Write     = 0
	v1Error     = 1
	v1Close     = 2
	v1Terminate = 1

	v1UtilitySession = 1
)

type v1Protocol struct {
	maxSession uint64
}

func newV1Protocol() *v1Protocol {
	return &v1Protocol{
		maxSession: 1,
	}
}

func (v *v1Protocol) onMessage(p protocolHandler, msg *Message) {
	if msg.Session == v1UtilitySession {
		v.dispatchUtilityMessage(p, msg)
		return
	}

	if v.maxSession < msg.Session {
		// It must be Invkoke
		if msg.MsgType != v1Invoke {
			fmt.Printf("new session %d must start from invoke type %d",
				msg.Session, msg.MsgType)
			return
		}

		v.maxSession = msg.Session
		p.onInvoke(msg)
		return
	}

	switch msg.MsgType {
	case v1Write:
		p.onChunk(msg)
	case v1Close:
		p.onChoke(msg)
	case v1Error:
		p.onError(msg)
	default:
		fmt.Printf("invalid message type: %d, message %v", msg.MsgType, msg)
	}
}

func (v *v1Protocol) isChunk(msg *Message) bool {
	return msg.MsgType == v1Write
}

func (v *v1Protocol) dispatchUtilityMessage(p protocolHandler, msg *Message) {
	switch msg.MsgType {
	case v1Heartbeat:
		p.onHeartbeat(msg)
	case v1Terminate:
		p.onTerminate(msg)
	}
}

func (v *v1Protocol) newHandshake(id string) *Message {
	return newHandshakeV1(id)
}

func (v *v1Protocol) newHeartbeat() *Message {
	return newHeartbeatV1()
}

func (v *v1Protocol) newChoke(session uint64) *Message {
	return newChokeV1(session)
}

func (v *v1Protocol) newChunk(session uint64, data []byte) *Message {
	return newChunkV1(session, data)
}

func (v *v1Protocol) newError(session uint64, code int, message string) *Message {
	return newErrorV1(session, code, message)
}

func newHandshakeV1(id string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: v1UtilitySession,
			MsgType: v1Handshake,
		},
		Payload: []interface{}{id},
	}
}

func newHeartbeatV1() *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: v1UtilitySession,
			MsgType: v1Heartbeat,
		},
		Payload: []interface{}{},
	}
}

func newInvokeV1(session uint64, event string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: invokeType,
		},
		Payload: []interface{}{event},
	}
}

func newChunkV1(session uint64, data []byte) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: v1Write,
		},
		Payload: []interface{}{data},
	}
}

func newErrorV1(session uint64, code int, message string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: v1Error,
		},
		Payload: []interface{}{code, message},
	}
}

func newChokeV1(session uint64) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: v1Close,
		},
		Payload: []interface{}{},
	}
}
