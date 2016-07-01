package cocaine12

import (
	"testing"
	"time"

	"golang.org/x/net/context"
)

func genBullets(sock *asyncRWSocket, bullets uint64) {
	for i := uint64(2); i < 2+bullets; i++ {
		sock.Write() <- newInvokeV1(i, "echo")
		sock.Write() <- newChunkV1(i, []byte("Dummy"))
		sock.Write() <- newChokeV1(i)
	}
}

func doBenchmarkWorkerEcho(b *testing.B, bullets uint64) {
	const (
		testID      = "uuid"
		testSession = 10
	)

	in, out := testConn()
	sock, _ := newAsyncRW(out)
	sock2, _ := newAsyncRW(in)
	w, err := newWorker(sock, testID, 1, true)
	if err != nil {
		panic(err)
	}

	w.disownTimer = time.NewTimer(1 * time.Hour)
	w.heartbeatTimer = time.NewTimer(1 * time.Hour)

	handlers := NewEventHandlers()
	handlers.On("echo", func(ctx context.Context, req Request, resp Response) {
		defer resp.Close()

		data, err := req.Read(ctx)
		if err != nil {
			panic(err)
		}
		resp.Write(data)
	})

	go func() {
		w.Run(handlers.Call, nil)
	}()
	defer w.Stop()

	genBullets(sock2, bullets)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := uint64(0); j < bullets; j++ {
			// chunk
			<-sock2.Read()
			// choke
			<-sock2.Read()
		}
	}
}

func BenchmarkWorkerEcho10(b *testing.B) {
	doBenchmarkWorkerEcho(b, 10)
}

func BenchmarkWorkerEcho100(b *testing.B) {
	doBenchmarkWorkerEcho(b, 100)
}

func BenchmarkWorkerEcho1000(b *testing.B) {
	doBenchmarkWorkerEcho(b, 1000)
}
