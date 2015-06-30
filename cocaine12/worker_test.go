package cocaine12

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
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
	w, err := newWorker(sock, testID, 0, true)
	if err != nil {
		t.Fatal("unable to create worker", err)
	}

	handlers := map[string]EventHandler{
		"test": func(ctx context.Context, req Request, res Response) {
			data, _ := req.Read()
			res.Write(data)
			res.Close()
		},
		"error": func(ctx context.Context, req Request, res Response) {
			_, _ = req.Read()
			res.ErrorMsg(-100, "dummyError")
		},
		"http": WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, method, r.Method)
			assert.Equal(t, "HTTP/"+version, r.Proto)
			assert.Equal(t, r.URL.String(), uri)
			assert.Equal(t, headersCocaineToHTTP(headers), r.Header)
			w.Header().Add("X-Test", "Test")
			w.WriteHeader(http.StatusProxyAuthRequired)
			fmt.Fprintf(w, "OK")
		}),
		"panic": func(ctx context.Context, req Request, res Response) {
			panic("PANIC")
		},
	}

	go func() {
		w.Run(handlers)
		close(onStop)
	}()

	corrupted := newInvokeV0(testSession+100, "AAA")
	corrupted.Payload = []interface{}{nil}
	sock2.Write() <- corrupted

	sock2.Write() <- newInvokeV0(testSession, "test")
	sock2.Write() <- newChunkV0(testSession, []byte("Dummy"))
	sock2.Write() <- newChokeV0(testSession)

	sock2.Write() <- newInvokeV0(testSession+1, "http")
	sock2.Write() <- newChunkV0(testSession+1, packTestReq(req))
	sock2.Write() <- newChokeV0(testSession + 1)

	sock2.Write() <- newInvokeV0(testSession+2, "error")
	sock2.Write() <- newChunkV0(testSession+2, []byte("Dummy"))
	sock2.Write() <- newChokeV0(testSession + 2)

	sock2.Write() <- newInvokeV0(testSession+3, "BadEvent")
	sock2.Write() <- newChunkV0(testSession+3, []byte("Dummy"))
	sock2.Write() <- newChokeV0(testSession + 3)

	sock2.Write() <- newInvokeV0(testSession+4, "panic")
	sock2.Write() <- newChunkV0(testSession+4, []byte("Dummy"))
	sock2.Write() <- newChokeV0(testSession + 4)

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
	assert.Equal(t, []byte("Dummy"), eChunk.Payload[0])
	eChoke := <-sock2.Read()
	checkTypeAndSession(t, eChoke, testSession, chokeType)

	// http event
	// status code & headers
	eChunk = <-sock2.Read()
	checkTypeAndSession(t, eChunk, testSession+1, chunkType)
	var firstChunk struct {
		Status  int
		Headers [][2]string
	}
	assert.NoError(t, testUnpackHttpChunk(eChunk.Payload, &firstChunk))
	assert.Equal(t, http.StatusProxyAuthRequired, firstChunk.Status, "http: invalid status code")
	assert.Equal(t, [][2]string{[2]string{"X-Test", "Test"}}, firstChunk.Headers, "http: headers")
	// body
	eChunk = <-sock2.Read()
	checkTypeAndSession(t, eChunk, testSession+1, chunkType)
	assert.Equal(t, eChunk.Payload[0].([]byte), []byte("OK"), "http: invalid body")
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
	w, err := newWorker(sock, testID, 0, true)
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

	sock2.Write() <- newHeartbeatV0()

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
