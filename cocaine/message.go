package cocaine

import (
	"bytes"
	"fmt"

	"github.com/ugorji/go/codec"
)

const (
	HANDSHAKE = iota
	HEARTBEAT
	TERMINATE
	INVOKE
	CHUNK
	ERROR
	CHOKE
)

type RawMessage []byte

var (
	mh codec.MsgpackHandle
	h  = &mh
)

type MessageInfo struct {
	Number  int64
	Channel int64
}

type Handshake struct {
	MessageInfo
	Uuid string
}

type Heartbeat struct {
	MessageInfo
}

type Terminate struct {
	MessageInfo
	Reason  string
	Message string
}

type Invoke struct {
	MessageInfo
	Event string
}

type Chunk struct {
	MessageInfo
	Data []byte
}

type ErrorMsg struct {
	MessageInfo
	Code    int
	Message string
}

type Choke struct {
	MessageInfo
}

type ServiceMethod struct {
	MessageInfo
	Data []interface{}
}

type Message interface {
	GetTypeID() int64
	GetSessionID() int64
	GetPayload() []interface{}
}

func (msg *MessageInfo) GetTypeID() int64 {
	return msg.Number
}

func (msg *MessageInfo) GetSessionID() int64 {
	return msg.Channel
}

func Pack(msg Message) RawMessage {
	var buf []byte
	err := codec.NewEncoderBytes(&buf, h).Encode([]interface{}{msg.GetTypeID(), msg.GetSessionID(), msg.GetPayload()})
	if err != nil {
		fmt.Println(err)
	}
	return buf
}

//ServiceMethod
func (msg *ServiceMethod) GetPayload() []interface{} {
	return msg.Data
}

// Handshake
func UnpackHandshake(session int64, data []interface{}) *Handshake {
	var uuid string
	if uuid_t, ok := data[0].([]byte); ok {
		uuid = string(uuid_t)
	}
	return &Handshake{MessageInfo{HANDSHAKE, session}, uuid}
}

func (msg *Handshake) GetPayload() []interface{} {
	return []interface{}{msg.Uuid}
}

// Heartbeat
func UnpackHeartbeat(session int64, data []interface{}) *Heartbeat {
	return &Heartbeat{MessageInfo{HEARTBEAT, session}}
}

func (msg *Heartbeat) GetPayload() []interface{} {
	return []interface{}{}
}

// Terminate
func UnpackTerminate(session int64, data []interface{}) *Terminate {
	var reason, message string
	if reason_t, ok := data[0].([]byte); ok {
		reason = string(reason_t)
	}
	if message_t, ok := data[1].([]byte); ok {
		message = string(message_t)
	}
	return &Terminate{MessageInfo{TERMINATE, session}, reason, message}
}

func (msg *Terminate) GetPayload() []interface{} {
	return []interface{}{msg.Reason, msg.Message}
}

// Invoke
func UnpackInvoke(session int64, data []interface{}) *Invoke {
	var event string
	if event_t, ok := data[0].([]byte); ok {
		event = string(event_t)
	} else {
		fmt.Println("Errror")
	}
	return &Invoke{MessageInfo{INVOKE, session}, event}
}

func (msg *Invoke) GetPayload() []interface{} {
	return []interface{}{msg.Event}
}

// Chunk
func UnpackChunk(session int64, data []interface{}) *Chunk {
	msg_data := data[0].([]byte)
	return &Chunk{MessageInfo{CHUNK, session}, msg_data}
}

func (msg *Chunk) GetPayload() []interface{} {
	return []interface{}{msg.Data}
}

// Error
func UnpackErrorMsg(session int64, data []interface{}) *ErrorMsg {
	var code int
	var message string
	if code_t, ok := data[0].(int); ok {
		code = code_t
	}
	if message_t, ok := data[1].([]byte); ok {
		message = string(message_t)
	}
	return &ErrorMsg{MessageInfo{ERROR, session}, code, message}
}

func (msg *ErrorMsg) GetPayload() []interface{} {
	return []interface{}{msg.Code, msg.Message}
}

// Choke
func UnpackChoke(session int64, data []interface{}) *Choke {
	return &Choke{MessageInfo{CHOKE, session}}
}

func (msg *Choke) GetPayload() []interface{} {
	return []interface{}{}
}

// Common unpacker
func UnpackMessage(input []interface{}) Message {
	defer func() {
		z := recover()
		if z != nil {
			fmt.Println("Error", z)
		}
	}()

	var session int64

	switch input[1].(type) {
	case uint64:
		session = int64(input[1].(uint64))
	case int64:
		session = input[1].(int64)
	}

	data := input[2].([]interface{})

	switch input[0].(int64) {
	case HANDSHAKE:
		return UnpackHandshake(session, data)
	case HEARTBEAT:
		return UnpackHeartbeat(session, data)
	case TERMINATE:
		return UnpackTerminate(session, data)
	case INVOKE:
		return UnpackInvoke(session, data)
	case CHUNK:
		return UnpackChunk(session, data)
	case ERROR:
		return UnpackErrorMsg(session, data)
	case CHOKE:
		return UnpackChoke(session, data)
	default:
		panic("Invalid message")
	}
}

type StreamUnpacker struct {
	buf []byte
}

func (unpacker *StreamUnpacker) Feed(data []byte) []Message {
	var msgs []Message
	unpacker.buf = append(unpacker.buf, data...)
	tmp := bytes.NewBuffer(unpacker.buf)
	dec := codec.NewDecoder(tmp, h)
	for {
		var res []interface{}
		err := dec.Decode(&res)
		if err != nil {
			break
		} else {
			msgs = append(msgs, UnpackMessage(res))
			unpacker.buf = unpacker.buf[len(unpacker.buf)-tmp.Len():]
		}

	}
	return msgs
}

func NewStreamUnpacker() *StreamUnpacker {
	return &StreamUnpacker{make([]byte, 0)}
}
