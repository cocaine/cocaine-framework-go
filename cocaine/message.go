package cocaine

import (
	"bytes"
	"fmt"

	"github.com/cocaine/cocaine-framework-go/pkg/github.com/ugorji/go/codec"
	uuid "github.com/satori/go.uuid"
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

	switch code_t := data[0].(type) {
	case uint64:
		code = int(code_t)
	case int64:
		code = int(code_t)
	}

	switch message_t := data[1].(type) {
	case []byte:
		message = string(message_t)
	case string:
		message = message_t
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

// Common unpacker
func unpackMessage(input []interface{}) (msg messageInterface, err error) {
	var session int64

	switch input[1].(type) {
	case uint64:
		session = int64(input[1].(uint64))
	case int64:
		session = input[1].(int64)
	}

	unpacker, ok := unpackers[input[0].(int64)]

	if !ok {
		err = fmt.Errorf("cocaine: invalid message type: %d", input[0].(int64))
	}

	data := input[2].([]interface{})
	msg, err = unpacker(session, data)

	return
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
			msg, err := unpackMessage(res)
			if err != nil {
				fmt.Printf("Error occured: %s", err)
				continue
			}
			msgs = append(msgs, msg)
			unpacker.buf = unpacker.buf[len(unpacker.buf)-tmp.Len():]
		}

	}
	return msgs
}

func newStreamUnpacker() *streamUnpacker {
	return &streamUnpacker{make([]byte, 0)}
}
