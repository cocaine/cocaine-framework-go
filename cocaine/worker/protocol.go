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
