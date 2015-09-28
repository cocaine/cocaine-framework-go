package cocaine12

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
)

const (
	traceId  = 80
	spanId   = 81
	parentId = 82
)

var (
	traceValueMap = map[uint64]struct{}{
		traceId:  struct{}{},
		spanId:   struct{}{},
		parentId: struct{}{},
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

// CommonMessageInfo consists of a session number and a message type
type CommonMessageInfo struct {
	// Session id
	Session uint64
	// Message type number
	MsgType uint64
}

func getTrace(header interface{}) (uint64, []byte, error) {
	switch t := header.(type) {
	case uint:
		return uint64(t), nil, nil
	case uint32:
		return uint64(t), nil, nil
	case uint64:
		return t, nil, nil
	case int:
		return uint64(t), nil, nil
	case int32:
		return uint64(t), nil, nil
	case int64:
		return uint64(t), nil, nil

	case []interface{}:
		if len(t) != 3 {
			return 0, nil, ErrInvalidHeaderLength
		}

		var (
			traceNum uint64
			traceVal []byte
		)

		switch num := t[1].(type) {
		case uint:
			traceNum = uint64(num)
		case uint32:
			traceNum = uint64(num)
		case uint64:
			traceNum = num
		case int:
			traceNum = uint64(num)
		case int32:
			traceNum = uint64(num)
		case int64:
			traceNum = uint64(num)
		default:
			fmt.Println(reflect.TypeOf(t[1]))
			return 0, nil, ErrInvalidTraceType
		}

		if _, ok := traceValueMap[traceNum]; !ok {
			return 0, nil, ErrInvalidTraceNumber
		}

		switch val := t[2].(type) {
		case []byte:
			traceVal = val
		case string:
			traceVal = []byte(val)
		default:
			return 0, nil, ErrInvalidTraceValueType
		}

		return traceNum, traceVal, nil
	default:
		fmt.Printf("%v\n", reflect.TypeOf(t))
	}

	return 0, nil, ErrInvalidHeaderType
}

type CocaineHeaders []interface{}

func (h CocaineHeaders) getTraceData() (traceInfo TraceInfo, err error) {
	var i = 0
	for _, header := range h {
		number, buffer, zerr := getTrace(header)
		if zerr != nil {
			continue
		}
		switch number {
		case traceId:
			if traceInfo.trace, err = decodeTracingId(buffer); err != nil {
				return
			}

		case spanId:
			if traceInfo.span, err = decodeTracingId(buffer); err != nil {
				return
			}

		case parentId:
			if buffer == nil {
				traceInfo.parent = 0
			} else {
				if traceInfo.parent, err = decodeTracingId(buffer); err != nil {
					return
				}
			}

		default:
			continue
		}

		i++
		if i == 3 {
			return
		}
	}

	return traceInfo, ErrNotAllTracesPresent
}

func decodeTracingId(b []byte) (uint64, error) {
	var tracingId uint64
	err := binary.Read(bytes.NewReader(b), binary.LittleEndian, &tracingId)
	return tracingId, err
}

func traceInfoToHeaders(info *TraceInfo) (CocaineHeaders, error) {
	var (
		offset  = 0
		buff    = new(bytes.Buffer)
		headers = make(CocaineHeaders, 0, 3)
	)

	if err := binary.Write(buff, binary.LittleEndian, info.trace); err != nil {
		return headers, err
	}
	headers = append(headers, []interface{}{false, traceId, buff.Bytes()[offset:]})
	offset = buff.Len()

	if err := binary.Write(buff, binary.LittleEndian, info.span); err != nil {
		return headers, err
	}
	headers = append(headers, []interface{}{false, spanId, buff.Bytes()[offset:]})
	offset = buff.Len()

	if err := binary.Write(buff, binary.LittleEndian, info.parent); err != nil {
		return headers, err
	}
	headers = append(headers, []interface{}{false, parentId, buff.Bytes()[offset:]})
	offset = buff.Len()

	return headers, nil
}

type Message struct {
	// _struct bool `codec:",toarray"`
	CommonMessageInfo
	Payload []interface{}
	Headers CocaineHeaders
}

func (m *Message) String() string {
	return fmt.Sprintf("message %v %v payload %v", m.MsgType, m.Session, m.Payload)
}
