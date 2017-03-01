package cocaine12

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tinylib/msgp/msgp"
	"github.com/ugorji/go/codec"
)

const (
	traceID  = 80
	spanID   = 81
	parentID = 82
)

var (
	traceValueMap = map[uint64]struct{}{
		traceID:  struct{}{},
		spanID:   struct{}{},
		parentID: struct{}{},
	}
)

var (
	ErrInvalidHeaderLength   = errors.New("invalid header size")
	ErrInvalidHeaderType     = errors.New("invalid header type")
	ErrInvalidTraceType      = errors.New("invalid trace header number type")
	ErrInvalidTraceNumber    = errors.New("invalid trace header number")
	ErrInvalidTraceValueType = errors.New("invalid trace value type")

	ErrNotAllTracesPresent = errors.New("not all trace values present")
)

// func getTrace(header interface{}) (uint64, []byte, error) {
// 	switch t := header.(type) {
// 	case uint:
// 		return uint64(t), nil, nil
// 	case uint32:
// 		return uint64(t), nil, nil
// 	case uint64:
// 		return t, nil, nil
// 	case int:
// 		return uint64(t), nil, nil
// 	case int32:
// 		return uint64(t), nil, nil
// 	case int64:
// 		return uint64(t), nil, nil
//
// 	case []interface{}:
// 		if len(t) != 3 {
// 			return 0, nil, ErrInvalidHeaderLength
// 		}
//
// 		var (
// 			traceNum uint64
// 			traceVal []byte
// 		)
//
// 		switch num := t[1].(type) {
// 		case uint:
// 			traceNum = uint64(num)
// 		case uint32:
// 			traceNum = uint64(num)
// 		case uint64:
// 			traceNum = num
// 		case int:
// 			traceNum = uint64(num)
// 		case int32:
// 			traceNum = uint64(num)
// 		case int64:
// 			traceNum = uint64(num)
// 		default:
// 			fmt.Println(reflect.TypeOf(t[1]))
// 			return 0, nil, ErrInvalidTraceType
// 		}
//
// 		if _, ok := traceValueMap[traceNum]; !ok {
// 			return 0, nil, ErrInvalidTraceNumber
// 		}
//
// 		switch val := t[2].(type) {
// 		case []byte:
// 			traceVal = val
// 		case string:
// 			traceVal = []byte(val)
// 		default:
// 			return 0, nil, ErrInvalidTraceValueType
// 		}
//
// 		return traceNum, traceVal, nil
// 	default:
// 		fmt.Printf("%v\n", reflect.TypeOf(t))
// 	}
//
// 	return 0, nil, ErrInvalidHeaderType
// }

// type CocaineHeaders []interface{}
// func (h CocaineHeaders) getTraceData() (traceInfo TraceInfo, err error) {
// 	var i = 0
// 	for _, header := range h {
// 		number, buffer, zerr := getTrace(header)
// 		if zerr != nil {
// 			continue
// 		}
// 		switch number {
// 		case traceId:
// 			if traceInfo.trace, err = decodeTracingId(buffer); err != nil {
// 				return
// 			}
//
// 		case spanId:
// 			if traceInfo.span, err = decodeTracingId(buffer); err != nil {
// 				return
// 			}
//
// 		case parentId:
// 			if buffer == nil {
// 				traceInfo.parent = 0
// 			} else {
// 				if traceInfo.parent, err = decodeTracingId(buffer); err != nil {
// 					return
// 				}
// 			}
//
// 		default:
// 			continue
// 		}
//
// 		i++
// 		if i == 3 {
// 			return
// 		}
// 	}
//
// 	return traceInfo, ErrNotAllTracesPresent
// }
//
// func decodeTracingId(b []byte) (uint64, error) {
// 	var tracingId uint64
// 	err := binary.Read(bytes.NewReader(b), binary.LittleEndian, &tracingId)
// 	return tracingId, err
// }
//
func traceInfoToHeaders(info *TraceInfo) (CocaineHeaders, error) {
	// var (
	// 	offset = 0
	// 	buff   = new(bytes.Buffer)
	// 	headers = make(CocaineHeaders, 0, 3)
	// )

	// if err := binary.Write(buff, binary.LittleEndian, info.trace); err != nil {
	// 	return headers, err
	// }
	// headers = append(headers, []interface{}{false, traceID, buff.Bytes()[offset:]})
	// offset = buff.Len()
	//
	// if err := binary.Write(buff, binary.LittleEndian, info.span); err != nil {
	// 	return headers, err
	// }
	// headers = append(headers, []interface{}{false, spanID, buff.Bytes()[offset:]})
	// offset = buff.Len()
	//
	// if err := binary.Write(buff, binary.LittleEndian, info.parent); err != nil {
	// 	return headers, err
	// }
	// headers = append(headers, []interface{}{false, parentID, buff.Bytes()[offset:]})
	//
	return nil, fmt.Errorf("NOT IMPLEMENTED")
}

const (
	minimalArraySize = 3
	sizeWithHeaders  = minimalArraySize + 1
)

var (
	// ErrInvalidPayloadType returns when a payload of encoded/decoded is not an ArrayType
	ErrInvalidPayloadType = errors.New("payload must be an ArrayType")
	// ErrInvalidMessageArraySize returns when the message is shorter than an array of size 3
	ErrInvalidMessageArraySize = errors.New("size of a message array less than 3")
)

var (
	mhAsocket = codec.MsgpackHandle{
		BasicHandle: codec.BasicHandle{
			EncodeOptions: codec.EncodeOptions{
				StructToArray: true,
			},
		},
	}
	hAsocket = &mhAsocket
)

// CocaineHeaders represents Cocaine HPACK like headers
type CocaineHeaders map[string][]string

func newCocaineHeaders() CocaineHeaders {
	return make(CocaineHeaders)
}

func (h CocaineHeaders) getTraceData() (TraceInfo, error) {
	return TraceInfo{}, fmt.Errorf("NOT IMPLEMENTED")
}

// EncodeMsg is needed to satisfy msgp.Encodable
func (h CocaineHeaders) EncodeMsg(w *msgp.Writer) error {
	// return fmt.Errorf("NOT IMPLEMENTED")
	return w.WriteArrayHeader(0)
}

// DecodeMsg is needed to satisfy msgp.Decodable
func (h *CocaineHeaders) DecodeMsg(r *msgp.Reader) error {
	// return fmt.Errorf("NOT IMPLEMENTED")
	return nil
}

// Message is a general Cocaine RPC message
type message struct {
	session uint64
	msgType uint64
	payload msgp.Raw
	headers CocaineHeaders
}

func newMessage(session, msgType uint64, args []interface{}, headers CocaineHeaders) (*message, error) {
	var buff = new(bytes.Buffer)
	// NOTE: using codec here to support old behavior. Also codec is more tolerant to a custom user types
	if err := codec.NewEncoder(buff, hAsocket).Encode(args); err != nil {
		return nil, err
	}
	return newMessageRawArgs(session, msgType, msgp.Raw(buff.Bytes()), headers), nil
}

func newMessageRawArgs(session, msgType uint64, payload msgp.Raw, headers CocaineHeaders) *message {
	return &message{
		session: session,
		msgType: msgType,
		payload: payload,
		headers: headers,
	}
}

// EncodeMsg is needed to satisfy msgp.Encodable
func (m *message) EncodeMsg(w *msgp.Writer) (err error) {
	// pack array header
	if err = w.WriteArrayHeader(sizeWithHeaders); err != nil {
		return err
	}
	// Pack session
	if err = w.WriteUint64(m.session); err != nil {
		return err
	}
	// Pack MsgType
	if err = w.WriteUint64(m.msgType); err != nil {
		return err
	}
	// Payload is already msgpacked
	// Check if it's an ArrayType and write it
	if nt := msgp.NextType(m.payload); nt != msgp.ArrayType {
		return err
	}
	if err = m.payload.EncodeMsg(w); err != nil {
		return err
	}
	// pack headers
	if err = m.headers.EncodeMsg(w); err != nil {
		return err
	}
	return nil
}

// DecodeMsg is needed to satisfy msgp.Decodable
func (m *message) DecodeMsg(r *msgp.Reader) error {
	sz, err := r.ReadArrayHeader()
	if err != nil {
		return err
	}
	if sz < minimalArraySize {
		return ErrInvalidMessageArraySize
	}

	var n msgp.Number
	// decode session
	if err = n.DecodeMsg(r); err != nil {
		return err
	}
	m.session, _ = n.Uint()

	// decode msgType (method)
	if err = n.DecodeMsg(r); err != nil {
		return err
	}
	m.msgType, _ = n.Uint()

	// payload
	if err = m.payload.DecodeMsg(r); err != nil {
		return err
	}

	m.headers = newCocaineHeaders()
	// decode headers if present
	if sz >= sizeWithHeaders {
		if err = m.headers.DecodeMsg(r); err != nil {
			return err
		}
		// Skip optional extra fields
		for ; sz > sizeWithHeaders; sz-- {
			if err = r.Skip(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *message) String() string {
	return fmt.Sprintf("message %d %d payload %s", m.msgType, m.session, m.payload)
}
