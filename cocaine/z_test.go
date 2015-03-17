package cocaine

import (
	"fmt"
	"io"
	"net/http"
	"testing"
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

func checkTypeAndSession(t *testing.T, msg *Message, eSession uint64, eType uint64) {
	if msg.MsgType != eType {
		t.Fatalf("%d expected, but got %d", eType, msg.MsgType)
	}
	if msg.Session != eSession {
		t.Fatal("Bad session number: %d instead of %d", msg.Session, eSession)
	}
}

func TestWorker(t *testing.T) {
	const (
		testId      = "uuid"
		testSession = 10
	)

	var (
		onStop = make(chan struct{})
	)

	in, out := testConn()
	sock, _ := NewAsyncRW(out)
	sock2, _ := NewAsyncRW(in)
	w, err := newWorker(sock, testId)
	if err != nil {
		t.Fatal("unable to create worker", err)
	}

	handlers := map[string]EventHandler{
		"test": func(req Request, res Response) {
			data := <-req.Read()
			res.Write(data)
			res.Close()
		},
		"http": WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "OK")
		}),
	}

	go func() {
		w.Run(handlers)
		close(onStop)
	}()

	sock2.Write() <- NewInvoke(testSession, "test")
	sock2.Write() <- NewChunk(testSession, "Dummy")
	sock2.Write() <- NewChoke(testSession)

	sock2.Write() <- NewInvoke(testSession+1, "http")
	sock2.Write() <- NewChunk(testSession+1, req)
	sock2.Write() <- NewChoke(testSession + 1)

	// handshake
	eHandshake := <-sock2.Read()
	checkTypeAndSession(t, eHandshake, 0, HandshakeType)

	switch uuid := eHandshake.Payload[0].(type) {
	case string:
		if uuid != testId {
			t.Fatal("bad uuid")
		}
	case []uint8:
		if string(uuid) != testId {
			t.Fatal("bad uuid")
		}
	default:
		t.Fatal("No uuid")
	}

	eHeartbeat := <-sock2.Read()
	checkTypeAndSession(t, eHeartbeat, 0, HeartbeatType)

	// test event
	eChunk := <-sock2.Read()
	checkTypeAndSession(t, eChunk, testSession, ChunkType)
	eChoke := <-sock2.Read()
	checkTypeAndSession(t, eChoke, testSession, ChokeType)

	// http event
	eChunk = <-sock2.Read()
	checkTypeAndSession(t, eChunk, testSession+1, ChunkType)
	eChunk = <-sock2.Read()
	checkTypeAndSession(t, eChunk, testSession+1, ChunkType)
	eChoke = <-sock2.Read()
	checkTypeAndSession(t, eChoke, testSession+1, ChokeType)
	<-onStop
}
