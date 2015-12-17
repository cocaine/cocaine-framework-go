package cocainetest

import (
	"bytes"
	"io"
	"syscall"

	"github.com/cocaine/cocaine-framework-go/cocaine12"
)

type Response struct {
	*bytes.Buffer
	closed bool
	Err    *CocaineError
}

type CocaineError struct {
	Msg  string
	Code int
}

var _ cocaine12.Response = NewResponse()

func NewResponse() *Response {
	return &Response{
		Buffer: new(bytes.Buffer),
		closed: false,
		Err:    nil,
	}
}

func (r *Response) Close() error {
	if r.closed {
		return syscall.EINVAL
	}

	r.closed = true
	return nil
}

func (r *Response) ZeroCopyWrite(data []byte) error {
	_, err := r.Buffer.Write(data)
	return err
}

func (r *Response) ErrorMsg(code int, msg string) error {
	if r.closed {
		return io.ErrClosedPipe
	}

	r.Err = &CocaineError{
		Msg:  msg,
		Code: code,
	}
	return r.Close()
}
