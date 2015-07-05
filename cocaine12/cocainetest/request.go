package cocainetest

import (
	"errors"
	"time"

	"github.com/cocaine/cocaine-framework-go/cocaine12"
)

var ErrNoChunks = errors.New("no chunks available")

type Request struct {
	chunks [][]byte
}

var _ cocaine12.Request = NewRequest()

func NewRequest() *Request {
	return &Request{
		chunks: make([][]byte, 10),
	}
}

func (r *Request) Push(chunk []byte) {
	r.chunks = append(r.chunks, chunk)
}

func (r *Request) Read(timeout ...time.Duration) (chunk []byte, err error) {
	if len(r.chunks) == 0 {
		return nil, ErrNoChunks
	}

	chunk, r.chunks = r.chunks[0], r.chunks[1:]
	return
}
