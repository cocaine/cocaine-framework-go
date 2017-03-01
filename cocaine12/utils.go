package cocaine12

import (
	"bytes"
	"context"
	"io"

	"github.com/tinylib/msgp/msgp"
	"github.com/ugorji/go/codec"
)

var (
	mPayloadHandler codec.MsgpackHandle
	payloadHandler  = &mPayloadHandler
)

func convertPayload(b msgp.Raw, out interface{}) error {
	return codec.NewDecoderBytes(b, payloadHandler).Decode(out)
}

type ReaderWithContext interface {
	io.Reader
	SetContext(ctx context.Context)
}

type requestReader struct {
	ctx    context.Context
	req    Request
	buffer *bytes.Buffer
}

func (r *requestReader) Read(p []byte) (int, error) {
	// If some data is available but not len(p) bytes,
	// Read conventionally returns what is available instead of waiting for more.
	if r.buffer.Len() > 0 {
		return r.buffer.Read(p)
	}

	data, err := r.req.Read(r.ctx)
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

func (r *requestReader) SetContext(ctx context.Context) {
	r.ctx = ctx
}

func RequestReader(ctx context.Context, req Request) ReaderWithContext {
	return &requestReader{
		ctx:    ctx,
		req:    req,
		buffer: new(bytes.Buffer),
	}
}
