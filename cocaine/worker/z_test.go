package worker

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/cocaine/cocaine-framework-go/cocaine/asio"
)

type pipeConn struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func (p *pipeConn) Read(b []byte) (int, error) {
	return p.reader.Read(b)
}
func (p *pipeConn) Write(b []byte) (int, error) {
	return p.writer.Write(b)
}
func (p *pipeConn) Close() error {
	p.reader.Close()
	return p.writer.Close()
}
func testConn() (io.ReadWriteCloser, io.ReadWriteCloser) {
	read1, write1 := io.Pipe()
	read2, write2 := io.Pipe()
	return &pipeConn{read1, write2}, &pipeConn{read2, write1}
}

func testHandler(req RequestStream, res ResponseStream) {
	fmt.Println("AAAA")
}

func TestWorker(t *testing.T) {
	in, out := testConn()
	sock, _ := asio.NewAsyncRW(out)
	sock2, _ := asio.NewAsyncRW(in)
	w, err := newWorker(sock, "test")
	if err != nil {
		t.Fatal("unable to create worker", err)
	}

	w.On("test", testHandler)
	go w.loop()
	sock2.Write() <- NewInvoke(1, "test")
	sock2.Write() <- NewChunk(1, "Dummy")
	sock2.Write() <- NewChoke(1)
	time.Sleep(time.Second * 5)
	w.Stop()
}
