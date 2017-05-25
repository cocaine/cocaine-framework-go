package cocaine12

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
		t.Fatalf("%d expected, but got %d: %v", eType, msg.MsgType, msg)
	}
	if msg.Session != eSession {
		t.Fatalf("Bad session number: %d instead of %d", msg.Session, eSession)
	}
}

func TestWorkerV1(t *testing.T) {
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
	w, err := newWorker(sock, testID, 1, true)
	if err != nil {
		t.Fatal("unable to create worker", err)
	}

	handlers := map[string]EventHandler{
		"test": func(ctx context.Context, req Request, res Response) {
			data, _ := req.Read(ctx)
			t.Logf("Request data: %s", data)
			res.Write(data)
			res.Close()
		},
		"error": func(ctx context.Context, req Request, res Response) {
			_, _ = req.Read(ctx)
			res.ErrorMsg(-100, "dummyError")
		},
		"http": WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, method, r.Method)
			assert.Equal(t, "HTTP/"+version, r.Proto)
			assert.Equal(t, r.URL.String(), uri)
			assert.Equal(t, HeadersCocaineToHTTP(headers), r.Header)
			w.Header().Add("X-Test", "Test")
			w.WriteHeader(http.StatusProxyAuthRequired)
			fmt.Fprint(w, "OK")
		}),
		"panic": func(ctx context.Context, req Request, res Response) {
			panic("PANIC")
		},
	}

	go func() {
		w.Run(handlers)
		close(onStop)
	}()

	corrupted := newInvokeV1(testSession-1, "AAA")
	corrupted.Payload = []interface{}{nil}
	sock2.Write() <- corrupted

	sock2.Write() <- newInvokeV1(testSession, "test")
	sock2.Write() <- newChunkV1(testSession, []byte("Dummy"))
	sock2.Write() <- newChokeV1(testSession)

	// handshake
	eHandshake, ok := <-sock2.Read()
	if eHandshake == nil {
		t.Fatalf("Corrupted message %s %v", eHandshake, ok)
	}
	checkTypeAndSession(t, eHandshake, v1UtilitySession, v1Handshake)

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
		t.Fatal("no uuid")
	}

	eHeartbeat := <-sock2.Read()
	checkTypeAndSession(t, eHeartbeat, v1UtilitySession, v1Heartbeat)

	// test event
	eChunk := <-sock2.Read()
	checkTypeAndSession(t, eChunk, testSession, v1Write)
	assert.Equal(t, []byte("Dummy"), eChunk.Payload[0])
	eChoke := <-sock2.Read()
	checkTypeAndSession(t, eChoke, testSession, v1Close)

	// http event
	// status code & headers
	t.Log("HTTP test:")
	sock2.Write() <- newInvokeV1(testSession+1, "http")
	sock2.Write() <- newChunkV1(testSession+1, packTestReq(req))
	sock2.Write() <- newChokeV1(testSession + 1)

	eChunk = <-sock2.Read()
	checkTypeAndSession(t, eChunk, testSession+1, v1Write)
	var firstChunk struct {
		Status  int
		Headers [][2]string
	}
	assert.NoError(t, testUnpackHTTPChunk(eChunk.Payload, &firstChunk))
	assert.Equal(t, http.StatusProxyAuthRequired, firstChunk.Status, "http: invalid status code")
	assert.Equal(t, [][2]string{[2]string{"X-Test", "Test"}}, firstChunk.Headers, "http: headers")
	// body
	eChunk = <-sock2.Read()
	checkTypeAndSession(t, eChunk, testSession+1, v1Write)
	assert.Equal(t, []byte("OK"), eChunk.Payload[0].([]byte), "http: invalid body %s", eChunk.Payload[0])
	eChoke = <-sock2.Read()
	checkTypeAndSession(t, eChoke, testSession+1, v1Close)

	// error event
	t.Log("error event")
	sock2.Write() <- newInvokeV1(testSession+2, "error")
	sock2.Write() <- newChunkV1(testSession+2, []byte("Dummy"))
	sock2.Write() <- newChokeV1(testSession + 2)

	eError := <-sock2.Read()
	checkTypeAndSession(t, eError, testSession+2, v1Error)

	// badevent
	t.Log("badevent event")
	sock2.Write() <- newInvokeV1(testSession+3, "BadEvent")
	sock2.Write() <- newChunkV1(testSession+3, []byte("Dummy"))
	sock2.Write() <- newChokeV1(testSession + 3)

	eError = <-sock2.Read()
	checkTypeAndSession(t, eError, testSession+3, v1Error)

	// panic
	t.Log("panic event")
	sock2.Write() <- newInvokeV1(testSession+4, "panic")
	sock2.Write() <- newChunkV1(testSession+4, []byte("Dummy"))
	sock2.Write() <- newChokeV1(testSession + 4)

	eError = <-sock2.Read()
	checkTypeAndSession(t, eError, testSession+4, v1Error)

	<-onStop
	w.Stop()
}

func TestWorkerV1Termination(t *testing.T) {
	const (
		testID = "uuid"
	)

	var onStop = make(chan struct{})

	in, out := testConn()
	sock, _ := newAsyncRW(out)
	sock2, _ := newAsyncRW(in)
	w, err := newWorker(sock, testID, 1, true)
	if err != nil {
		t.Fatal("unable to create worker", err)
	}

	go func() {
		w.Run(map[string]EventHandler{})
		close(onStop)
	}()

	eHandshake, ok := <-sock2.Read()
	if eHandshake == nil {
		t.Fatalf("Corrupted message %s %v", eHandshake, ok)
	}
	checkTypeAndSession(t, eHandshake, v1UtilitySession, v1Handshake)
	eHeartbeat := <-sock2.Read()
	checkTypeAndSession(t, eHeartbeat, v1UtilitySession, v1Heartbeat)

	sock2.Write() <- newHeartbeatV1()

	terminate := &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: v1UtilitySession,
			MsgType: v1Terminate,
		},
		Payload: []interface{}{100, "TestTermination"},
	}

	corrupted := &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: v1UtilitySession,
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
		checkTypeAndSession(t, eHeartbeat, v1UtilitySession, v1Heartbeat)
	}

	sock2.Write() <- terminate

	select {
	case <-onStop:
		// a termination exit
	case <-time.After(disownTimeout):
		t.Fatalf("unexpected exit")
	}
}

func TestWorkerLoad(t *testing.T) {
	const (
		testID = "uuid"
		limit  = 10240
	)

	var (
		onStop = make(chan struct{})
	)

	in, out := testConn()
	sock, _ := newAsyncRW(out)
	sock2, _ := newAsyncRW(in)
	w, err := newWorker(sock, testID, 1, true)
	if err != nil {
		t.Fatal("unable to create worker", err)
	}

	w.On("echo", func(ctx context.Context, req Request, res Response) {
		defer res.Close()
		req.Read(ctx)
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		res.Write([]byte("DONE"))
	})

	go func() {
		w.Run(map[string]EventHandler{})
	}()

	j := 0
	sessions := 0
	go func() {
		for {
			select {
			case msg := <-sock2.Read():
				switch {
				case msg == nil:
					t.Log(j)
					t.FailNow()
				case msg.Session == v1UtilitySession:
					sock2.Write() <- newHeartbeatV1()
				default:
					j++
					if msg.MsgType == v1Close {
						sessions++
						if sessions == limit-1 {
							w.Stop()
							t.Log("Done")
							close(onStop)
							return
						}
					}
				}
			}
		}
	}()

	i := uint64(1)

	for k := 0; k < 5; k++ {
		func() {
			for {
				val := atomic.AddUint64(&i, 1)
				if val > limit {
					return
				}
				sock2.Write() <- newInvokeV1(val, "echo")
				sock2.Write() <- newChunkV1(val, []byte("Dummy"))
				sock2.Write() <- newChokeV1(val)

			}
		}()
	}

	select {
	case <-onStop:
		// a termination exit
		t.Logf("messages %d, sessions %d, limit %d\n", j, sessions, limit-1)
	case <-time.After(time.Second * 100):
		t.Fatalf("unexpected exit")
	}
}
