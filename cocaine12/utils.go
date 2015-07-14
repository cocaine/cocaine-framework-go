package cocaine12

import (
	"bytes"
	"io"

	"github.com/cocaine/cocaine-framework-go/vendor/src/github.com/ugorji/go/codec"
)

var (
	mPayloadHandler codec.MsgpackHandle
	payloadHandler  = &mPayloadHandler
)

func convertPayload(in interface{}, out interface{}) error {
	var buf []byte
	if err := codec.NewEncoderBytes(&buf, payloadHandler).Encode(in); err != nil {
		return err
	}
	if err := codec.NewDecoderBytes(buf, payloadHandler).Decode(out); err != nil {
		return err
	}
	return nil
}

type requestReader struct {
	req    Request
	buffer *bytes.Buffer
}

func (r *requestReader) Read(p []byte) (int, error) {
	// If some data is available but not len(p) bytes,
	// Read conventionally returns what is available instead of waiting for more.
	if r.buffer.Len() > 0 {
		return r.buffer.Read(p)
	}

	data, err := r.req.Read()
	switch err {
	case nil:
		// copy the current data to a provided []byte directly
		n := copy(p, data)
		// if not all the data were copied
		// put the rest into a buffer
		if n < len(data) {
			r.buffer.Write(data[n:])
		}
		return n, nil

	case ErrStreamIsClosed:
		return 0, io.EOF

	default:
		return 0, err
	}
}

func RequestReader(req Request) io.Reader {
	return &requestReader{
		req:    req,
		buffer: new(bytes.Buffer),
	}
}
