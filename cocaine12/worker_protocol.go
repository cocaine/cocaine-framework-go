package cocaine12

const (
	v0 = 0
	v1 = 1
)

type messageTypeDetector interface {
	isChunk(msg *Message) bool
}

type protocolHandler interface {
	onChoke(msg *Message)
	onChunk(msg *Message)
	onError(msg *Message)
	onHeartbeat(msg *Message)
	onInvoke(msg *Message)
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
	newError(session uint64, code int, message string) *Message
}

type protocolDispather interface {
	utilityProtocolGenerator
	handlerProtocolGenerator
	onMessage(p protocolHandler, msg *Message)
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
