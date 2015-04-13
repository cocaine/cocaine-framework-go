package cocaine12

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
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
		testID      = "uuid"
		testSession = 10
	)

	var (
		onStop = make(chan struct{})
	)

	in, out := testConn()
	sock, _ := newAsyncRW(out)
	sock2, _ := newAsyncRW(in)
	w, err := newWorker(sock, testID)
	if err != nil {
		t.Fatal("unable to create worker", err)
	}

	handlers := map[string]EventHandler{
		"test": func(req Request, res Response) {
			data, _ := req.Read()
			res.Write(data)
			res.Close()
		},
		"error": func(req Request, res Response) {
			_, _ = req.Read()
			res.ErrorMsg(-100, "dummyError")
		},
		"http": WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "OK")
		}),
		"panic": func(req Request, res Response) {
			panic("PANIC")
		},
	}

	go func() {
		w.Run(handlers)
		close(onStop)
	}()

	corrupted := newInvoke(testSession+100, "AAA")
	corrupted.Payload = []interface{}{nil}
	sock2.Write() <- corrupted

	sock2.Write() <- newInvoke(testSession, "test")
	sock2.Write() <- newChunk(testSession, "Dummy")
	sock2.Write() <- newChoke(testSession)

	sock2.Write() <- newInvoke(testSession+1, "http")
	sock2.Write() <- newChunk(testSession+1, req)
	sock2.Write() <- newChoke(testSession + 1)

	sock2.Write() <- newInvoke(testSession+2, "error")
	sock2.Write() <- newChunk(testSession+2, "Dummy")
	sock2.Write() <- newChoke(testSession + 2)

	sock2.Write() <- newInvoke(testSession+3, "BadEvent")
	sock2.Write() <- newChunk(testSession+3, "Dummy")
	sock2.Write() <- newChoke(testSession + 3)

	sock2.Write() <- newInvoke(testSession+4, "panic")
	sock2.Write() <- newChunk(testSession+4, "Dummy")
	sock2.Write() <- newChoke(testSession + 4)

	// handshake
	eHandshake := <-sock2.Read()
	checkTypeAndSession(t, eHandshake, 0, handshakeType)

	switch uuid := eHandshake.Payload[0].(type) {
	case string:
		if uuid != testID {
			t.Fatal("bad uuid")
		}
	case []uint8:
		if string(uuid) != testID {
			t.Fatal("bad uuid")
		}
	default:
		t.Fatal("No uuid")
	}

	eHeartbeat := <-sock2.Read()
	checkTypeAndSession(t, eHeartbeat, 0, heartbeatType)

	// test event
	eChunk := <-sock2.Read()
	checkTypeAndSession(t, eChunk, testSession, chunkType)
	eChoke := <-sock2.Read()
	checkTypeAndSession(t, eChoke, testSession, chokeType)

	// http event
	eChunk = <-sock2.Read()
	checkTypeAndSession(t, eChunk, testSession+1, chunkType)
	eChunk = <-sock2.Read()
	checkTypeAndSession(t, eChunk, testSession+1, chunkType)
	eChoke = <-sock2.Read()
	checkTypeAndSession(t, eChoke, testSession+1, chokeType)

	// error event
	eError := <-sock2.Read()
	checkTypeAndSession(t, eError, testSession+2, errorType)
	eChoke = <-sock2.Read()
	checkTypeAndSession(t, eChoke, testSession+2, chokeType)

	// badevent
	eError = <-sock2.Read()
	checkTypeAndSession(t, eError, testSession+3, errorType)
	eChoke = <-sock2.Read()
	checkTypeAndSession(t, eChoke, testSession+3, chokeType)

	// panic
	eError = <-sock2.Read()
	checkTypeAndSession(t, eError, testSession+4, errorType)
	eChoke = <-sock2.Read()
	checkTypeAndSession(t, eChoke, testSession+4, chokeType)
	<-onStop
	w.Stop()
}

func TestWorkerTermination(t *testing.T) {
	const (
		testID = "uuid"
	)

	var onStop = make(chan struct{})

	in, out := testConn()
	sock, _ := newAsyncRW(out)
	sock2, _ := newAsyncRW(in)
	w, err := newWorker(sock, testID)
	if err != nil {
		t.Fatal("unable to create worker", err)
	}

	go func() {
		w.Run(map[string]EventHandler{})
		close(onStop)
	}()

	eHandshake := <-sock2.Read()
	checkTypeAndSession(t, eHandshake, 0, handshakeType)
	eHeartbeat := <-sock2.Read()
	checkTypeAndSession(t, eHeartbeat, 0, heartbeatType)

	sock2.Write() <- newHeartbeatMessage()

	terminate := &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: 0,
			MsgType: terminateType,
		},
		Payload: []interface{}{100, "TestTermination"},
	}

	corrupted := &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: 0,
			MsgType: 9999,
		},
		Payload: []interface{}{100, "TestTermination"},
	}

	sock2.Write() <- corrupted

	select {
	case <-onStop:
		// an unexpected disown exit
		t.Fatalf("unexpected exit")
	case <-time.After(heartbeatTimeout + time.Second):
		t.Fatalf("unexpected timeout")
	case eHeartbeat := <-sock2.Read():
		checkTypeAndSession(t, eHeartbeat, 0, heartbeatType)
	}

	sock2.Write() <- terminate

	select {
	case <-onStop:
		// a termination exit
	case <-time.After(disownTimeout):
		t.Fatalf("unexpected exit")
	}
}
