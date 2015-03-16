package cocaine

import (
	// "bytes"
	"fmt"

	uuid "github.com/satori/go.uuid"
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

type messageInfo struct {
	Channel uint64
	Number  int64
}

type handshakeStruct struct {
	messageInfo
	Uuid uuid.UUID
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
	getSessionID() uint64
	getPayload() []interface{}
}

func (msg *messageInfo) getTypeID() int64 {
	return msg.Number
}

func (msg *messageInfo) getSessionID() uint64 {
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
func unpackHandshake(session int64, data []interface{}) (msg messageInterface, err error) {
	u := uuid.UUID{}
	if uuid_t, ok := data[0].([]byte); ok {
		u, err = uuid.FromString(string(uuid_t))
		if err != nil {
			return
		}
	}
	msg = &handshakeStruct{messageInfo{HANDSHAKE, session}, u}
	return
}

func (msg *handshakeStruct) getPayload() []interface{} {
	return []interface{}{msg.Uuid.String()}
}

// heartbeat
func unpackHeartbeat(session int64, data []interface{}) (msg messageInterface, err error) {
	msg = &heartbeat{messageInfo{HEARTBEAT, session}}
	return
}

func (msg *heartbeat) getPayload() []interface{} {
	return []interface{}{}
}

// terminateStruct
func unpackTerminate(session int64, data []interface{}) (msg messageInterface, err error) {
	var reason, message string
	if reason_t, ok := data[0].([]byte); ok {
		reason = string(reason_t)
	}
	if message_t, ok := data[1].([]byte); ok {
		message = string(message_t)
	}
	msg = &terminateStruct{messageInfo{TERMINATE, session}, reason, message}
	return
}

func (msg *terminateStruct) getPayload() []interface{} {
	return []interface{}{msg.Reason, msg.Message}
}

// invoke
func unpackInvoke(session int64, data []interface{}) (msg messageInterface, err error) {
	var event string
	if event_t, ok := data[0].([]byte); ok {
		event = string(event_t)
	} else {
		fmt.Println("Errror")
	}
	msg = &invoke{messageInfo{INVOKE, session}, event}
	return
}

func (msg *invoke) getPayload() []interface{} {
	return []interface{}{msg.Event}
}

// chunk
func unpackChunk(session int64, data []interface{}) (msg messageInterface, err error) {
	msgData := data[0].([]byte)
	msg = &chunk{messageInfo{CHUNK, session}, msgData}
	return
}

func (msg *chunk) getPayload() []interface{} {
	return []interface{}{msg.Data}
}

// Error
func unpackErrorMsg(session int64, data []interface{}) (msg messageInterface, err error) {
	var code int
	var message string
	if code_t, ok := data[0].(int); ok {
		code = code_t
	}
	if message_t, ok := data[1].([]byte); ok {
		message = string(message_t)
	}
	msg = &errorMsg{messageInfo{ERROR, session}, code, message}
	return
}

func (msg *errorMsg) getPayload() []interface{} {
	return []interface{}{msg.Code, msg.Message}
}

// choke
func unpackChoke(session int64, data []interface{}) (msg messageInterface, err error) {
	msg = &choke{messageInfo{CHOKE, session}}
	return
}

func (msg *choke) getPayload() []interface{} {
	return []interface{}{}
}

type unpacker func(int64, []interface{}) (messageInterface, error)

var unpackers = map[int64]unpacker{
	HANDSHAKE: unpackHandshake,
	HEARTBEAT: unpackHeartbeat,
	TERMINATE: unpackTerminate,
	INVOKE:    unpackInvoke,
	CHUNK:     unpackChunk,
	ERROR:     unpackErrorMsg,
	CHOKE:     unpackChoke,
}
