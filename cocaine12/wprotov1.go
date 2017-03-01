package cocaine12

import (
	"fmt"

	"github.com/tinylib/msgp/msgp"
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

func newV1Protocol() protocolDispather {
	return &v1Protocol{
		maxSession: 1,
	}
}

func (v *v1Protocol) onMessage(p protocolHandler, msg *message) error {
	if msg.session == v1UtilitySession {
		return v.dispatchUtilityMessage(p, msg)
	}

	if v.maxSession < msg.session {
		// It must be Invkoke
		if msg.msgType != v1Invoke {
			return fmt.Errorf("new session %d must start from invoke type %d, not %d\n",
				msg.session, v1Invoke, msg.msgType)
		}

		v.maxSession = msg.session
		return p.onInvoke(msg)
	}

	switch msg.msgType {
	case v1Write:
		p.onChunk(msg)
	case v1Close:
		p.onChoke(msg)
	case v1Error:
		p.onError(msg)
	default:
		return fmt.Errorf("an invalid message type: %d, message %v", msg.msgType, msg)
	}
	return nil
}

func (v *v1Protocol) isChunk(msg *message) bool {
	return msg.msgType == v1Write
}

func (v *v1Protocol) dispatchUtilityMessage(p protocolHandler, msg *message) error {
	switch msg.msgType {
	case v1Heartbeat:
		p.onHeartbeat(msg)
	case v1Terminate:
		p.onTerminate(msg)
	default:
		return fmt.Errorf("an invalid utility message type %d", msg.msgType)
	}

	return nil
}

func (v *v1Protocol) newHandshake(id string) *message {
	return newHandshakeV1(id)
}

func (v *v1Protocol) newHeartbeat() *message {
	return newHeartbeatV1()
}

func (v *v1Protocol) newChoke(session uint64) *message {
	return newChokeV1(session)
}

func (v *v1Protocol) newChunk(session uint64, data []byte) *message {
	return newChunkV1(session, data)
}

func (v *v1Protocol) newError(session uint64, category, code int, msg string) *message {
	return newErrorV1(session, category, code, msg)
}

func newHandshakeV1(id string) *message {
	payload := msgp.AppendArrayHeader(nil, 1)
	payload = msgp.AppendString(payload, id)
	return newMessageRawArgs(v1UtilitySession, v1Handshake, payload, nil)
}

func newHeartbeatV1() *message {
	payload := msgp.AppendArrayHeader(nil, 0)
	return newMessageRawArgs(v1UtilitySession, v1Heartbeat, payload, nil)
}

func newInvokeV1(session uint64, event string) *message {
	payload := msgp.AppendArrayHeader(nil, 1)
	payload = msgp.AppendString(payload, event)
	return newMessageRawArgs(session, v1Invoke, payload, nil)
}

func newChunkV1(session uint64, data []byte) *message {
	payload := msgp.AppendArrayHeader(nil, 1)
	payload = msgp.AppendStringFromBytes(payload, data)
	return newMessageRawArgs(session, v1Write, payload, nil)
}

func newErrorV1(session uint64, category, code int, message string) *message {
	payload := msgp.AppendArrayHeader(nil, 2)
	// pack category and code
	payload = msgp.AppendArrayHeader(payload, 2)
	payload = msgp.AppendInt(payload, category)
	payload = msgp.AppendInt(payload, code)
	// pack error message
	payload = msgp.AppendString(payload, message)
	return newMessageRawArgs(session, v1Error, payload, nil)
}

func newChokeV1(session uint64) *message {
	payload := msgp.AppendArrayHeader(nil, 0)
	return newMessageRawArgs(session, v1Close, payload, nil)
}
