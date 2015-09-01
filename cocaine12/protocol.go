package cocaine12

import (
	"fmt"
)

// CommonMessageInfo consists of a session number and a message type
type CommonMessageInfo struct {
	// Session id
	Session uint64
	// Message type number
	MsgType uint64
}

type Message struct {
	CommonMessageInfo
	Payload []interface{}
}

func (m *Message) String() string {
	return fmt.Sprintf("message %v %v payload %v", m.MsgType, m.Session, m.Payload)
}
