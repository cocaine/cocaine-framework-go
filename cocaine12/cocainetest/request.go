package cocainetest

import (
	"context"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
)

type Request struct {
	chunks [][]byte
}

var _ cocaine.Request = NewRequest()

func NewRequest() *Request {
	return &Request{
		chunks: make([][]byte, 0, 10),
	}
}

func (r *Request) Write(p []byte) (int, error) {
	r.chunks = append(r.chunks, p)
	return len(p), nil
}

func (r *Request) Read(ctx context.Context) (chunk []byte, err error) {
	if len(r.chunks) == 0 {
		return nil, cocaine.ErrStreamIsClosed
	}

	chunk, r.chunks = r.chunks[0], r.chunks[1:]
	return
}
