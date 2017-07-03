package cocaine12

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/tinylib/msgp/msgp"
	"github.com/ugorji/go/codec"
)

const (
	traceID  = "trace_id"
	spanID   = "span_id"
	parentID = "parent_id"
)

func decodeTracingID(s string) (uint64, error) {
	var tracingID uint64
	err := binary.Read(strings.NewReader(s), binary.LittleEndian, &tracingID)
	return tracingID, err
}

func traceInfoToHeaders(h CocaineHeaders, info *TraceInfo) {
	// TODO: sync.Pool
	var buff = new(bytes.Buffer)

	// NOTE: binary.Write may return an error only from Writer.Write call
	// As buff wouldn't return it, we can omit the error check

	// Pack TraceID
	binary.Write(buff, binary.LittleEndian, info.trace)
	h[traceID] = append(h[traceID], buff.String())
	buff.Reset()

	// Pack SpanID
	binary.Write(buff, binary.LittleEndian, info.span)
	h[spanID] = append(h[spanID], buff.String())
	buff.Reset()

	// Pack ParentID
	binary.Write(buff, binary.LittleEndian, info.parent)
	h[parentID] = append(h[parentID], buff.String())
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

	ErrNotAllTracesPresent = errors.New("not all trace values present")
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

func (h CocaineHeaders) getTraceData() (tr TraceInfo, err error) {
	if v := h[traceID]; len(v) > 0 {
		if tr.trace, err = decodeTracingID(v[0]); err != nil {
			return tr, fmt.Errorf("failed to decode TraceID: %v", err)
		}
	} else {
		return tr, ErrNotAllTracesPresent
	}

	if v := h[spanID]; len(v) > 0 {
		if tr.span, err = decodeTracingID(v[0]); err != nil {
			return tr, fmt.Errorf("failed to decode SpanID: %v", err)
		}
	} else {
		return tr, ErrNotAllTracesPresent
	}

	if v := h[parentID]; len(v) > 0 {
		if tr.parent, err = decodeTracingID(v[0]); err != nil {
			return tr, fmt.Errorf("failed to decode ParentID: %v", err)
		}
	} else {
		return tr, ErrNotAllTracesPresent
	}

	return tr, nil
}

// EncodeMsg is needed to satisfy msgp.Encodable
func (h CocaineHeaders) EncodeMsg(w *msgp.Writer) (err error) {
	if len(h) == 0 {
		return nil
	}
	var sz uint32
	// TODO: sync.Pool
	var b []byte
	for k, v := range h {
		for _, item := range v {
			sz++
			// false key value
			b = msgp.AppendArrayHeader(b, 3)
			b = msgp.AppendBool(b, false)
			b = msgp.AppendString(b, k)
			b = msgp.AppendString(b, item)
			msgp.NewWriter(w)
		}
	}
	if err = w.WriteArrayHeader(sz); err != nil {
		return err
	}
	return w.Append(b...)
}

// DecodeMsg is needed to satisfy msgp.Decodable
func (h CocaineHeaders) DecodeMsg(r *msgp.Reader) error {
	var sz uint32
	var err error
	if sz, err = r.ReadArrayHeader(); err != nil {
		return fmt.Errorf("failed to read ArrayHeader for headers: %v", err)
	}
	// NOTE: it does not support HPACK dynamic table
	// We assume that all headers arrive as string keys.
	// Non-supported headers just skipped

	var hsz uint32
	var headerType msgp.Type
	for i := uint32(0); i < sz; i++ {
		if headerType, err = r.NextType(); err != nil {
			return fmt.Errorf("failed to get msgp.Type for a header: %v", err)
		}
		switch headerType {
		case msgp.IntType, msgp.UintType:
			// skip headers for HPACK table
			if err = r.Skip(); err != nil {
				return fmt.Errorf("failed to Skip unsupported header with HPACK table: %v", err)
			}

		case msgp.ArrayType:
			if hsz, err = r.ReadArrayHeader(); err != nil {
				return fmt.Errorf("failed to read ArrayHeader for a header: %v", err)
			}
			if hsz != 3 {
				return fmt.Errorf("invalid header array length %d", hsz)
			}
			if _, err = r.ReadBool(); err != nil {
				return fmt.Errorf("failed to get Bool flag for a header: %v", err)
			}
			var headerKeyType msgp.Type
			headerKeyType, err = r.NextType()
			if err != nil {
				return fmt.Errorf("failed to get NextType for a header name: %v", err)
			}
			switch headerKeyType {
			case msgp.IntType, msgp.UintType:
				// Skip header key
				r.Skip()
				// Skip header value
				r.Skip()
			case msgp.BinType, msgp.StrType:
				var key string
				if key, err = r.ReadString(); err != nil {
					return fmt.Errorf("failed to read header key: %v", err)
				}
				var value string
				if value, err = r.ReadString(); err != nil {
					return fmt.Errorf("failed to read header value: %v", err)
				}
				h[key] = append(h[key], value)
			default:
				return fmt.Errorf("unsupported header key key: %s", headerKeyType)
			}
		}
	}

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

	// decode headers if present
	if sz >= sizeWithHeaders {
		m.headers = make(CocaineHeaders)
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
