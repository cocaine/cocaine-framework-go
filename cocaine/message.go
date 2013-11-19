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

type rawMessage []byte

var (
	mh codec.MsgpackHandle
	h  = &mh
)

type messageInfo struct {
	Number  int64
	Channel int64
}

type handshakeStruct struct {
	messageInfo
	Uuid string
}

type heartbeat struct {
	messageInfo
}

type terminateStruct struct {
	messageInfo
	Reason  string
	Message string
}

type invoke struct {
	messageInfo
	Event string
}

type chunk struct {
	messageInfo
	Data []byte
}

type errorMsg struct {
	messageInfo
	Code    int
	Message string
}

type choke struct {
	messageInfo
}

type ServiceMethod struct {
	messageInfo
	Data []interface{}
}

type messageInterface interface {
	getTypeID() int64
	getSessionID() int64
	getPayload() []interface{}
}

func (msg *messageInfo) getTypeID() int64 {
	return msg.Number
}

func (msg *messageInfo) getSessionID() int64 {
	return msg.Channel
}

func packMsg(msg messageInterface) rawMessage {
	var buf []byte
	err := codec.NewEncoderBytes(&buf, h).Encode([]interface{}{msg.getTypeID(), msg.getSessionID(), msg.getPayload()})
	if err != nil {
		fmt.Println(err)
	}
	return buf
}

//ServiceMethod
func (msg *ServiceMethod) getPayload() []interface{} {
	return msg.Data
}

// handshakeStruct
func unpackHandshake(session int64, data []interface{}) *handshakeStruct {
	var uuid string
	if uuid_t, ok := data[0].([]byte); ok {
		uuid = string(uuid_t)
	}
	return &handshakeStruct{messageInfo{HANDSHAKE, session}, uuid}
}

func (msg *handshakeStruct) getPayload() []interface{} {
	return []interface{}{msg.Uuid}
}

// heartbeat
func unpackHeartbeat(session int64, data []interface{}) *heartbeat {
	return &heartbeat{messageInfo{HEARTBEAT, session}}
}

func (msg *heartbeat) getPayload() []interface{} {
	return []interface{}{}
}

// terminateStruct
func unpackTerminate(session int64, data []interface{}) *terminateStruct {
	var reason, message string
	if reason_t, ok := data[0].([]byte); ok {
		reason = string(reason_t)
	}
	if message_t, ok := data[1].([]byte); ok {
		message = string(message_t)
	}
	return &terminateStruct{messageInfo{TERMINATE, session}, reason, message}
}

func (msg *terminateStruct) getPayload() []interface{} {
	return []interface{}{msg.Reason, msg.Message}
}

// invoke
func unpackInvoke(session int64, data []interface{}) *invoke {
	var event string
	if event_t, ok := data[0].([]byte); ok {
		event = string(event_t)
	} else {
		fmt.Println("Errror")
	}
	return &invoke{messageInfo{INVOKE, session}, event}
}

func (msg *invoke) getPayload() []interface{} {
	return []interface{}{msg.Event}
}

// chunk
func unpackChunk(session int64, data []interface{}) *chunk {
	msg_data := data[0].([]byte)
	return &chunk{messageInfo{CHUNK, session}, msg_data}
}

func (msg *chunk) getPayload() []interface{} {
	return []interface{}{msg.Data}
}

// Error
func unpackErrorMsg(session int64, data []interface{}) *errorMsg {
	var code int
	var message string
	if code_t, ok := data[0].(int); ok {
		code = code_t
	}
	if message_t, ok := data[1].([]byte); ok {
		message = string(message_t)
	}
	return &errorMsg{messageInfo{ERROR, session}, code, message}
}

func (msg *errorMsg) getPayload() []interface{} {
	return []interface{}{msg.Code, msg.Message}
}

// choke
func unpackChoke(session int64, data []interface{}) *choke {
	return &choke{messageInfo{CHOKE, session}}
}

func (msg *choke) getPayload() []interface{} {
	return []interface{}{}
}

// Common unpacker
func unpackMessage(input []interface{}) messageInterface {
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
		return unpackHandshake(session, data)
	case HEARTBEAT:
		return unpackHeartbeat(session, data)
	case TERMINATE:
		return unpackTerminate(session, data)
	case INVOKE:
		return unpackInvoke(session, data)
	case CHUNK:
		return unpackChunk(session, data)
	case ERROR:
		return unpackErrorMsg(session, data)
	case CHOKE:
		return unpackChoke(session, data)
	default:
		panic("Invalid message")
	}
}

type streamUnpacker struct {
	buf []byte
}

func (unpacker *streamUnpacker) Feed(data []byte) []messageInterface {
	var msgs []messageInterface
	unpacker.buf = append(unpacker.buf, data...)
	tmp := bytes.NewBuffer(unpacker.buf)
	dec := codec.NewDecoder(tmp, h)
	for {
		var res []interface{}
		err := dec.Decode(&res)
		if err != nil {
			break
		} else {
			msgs = append(msgs, unpackMessage(res))
			unpacker.buf = unpacker.buf[len(unpacker.buf)-tmp.Len():]
		}

	}
	return msgs
}

func newStreamUnpacker() *streamUnpacker {
	return &streamUnpacker{make([]byte, 0)}
}
