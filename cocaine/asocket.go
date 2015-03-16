package cocaine

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/ugorji/go/codec"
)

var (
	mhAsocket = codec.MsgpackHandle{
		BasicHandle: codec.BasicHandle{
			EncodeOptions: codec.EncodeOptions{
				StructToArray: true,
			},
		},
	}
	hAsocket = &mhAsocket
)

type SocketIO interface {
	Read() chan *Message
	Write() chan *Message
	IsClosed() <-chan struct{}
	Close()
}

type asyncBuff struct {
	wg      sync.WaitGroup
	in, out chan *Message
	// ToDO: it should be chan time.Duration
	// to implement the drain method with a timeout
	stop chan struct{}
}

func newAsyncBuf() *asyncBuff {
	buf := &asyncBuff{
		in:   make(chan *Message),
		out:  make(chan *Message),
		stop: make(chan struct{})}

	buf.loop()
	return buf
}

func (bf *asyncBuff) loop() {
	bf.wg.Add(1)
	go func() {
		defer bf.wg.Done()

		var (
			pending []*Message            // data buffer
			_in     chan *Message = bf.in // incoming channel
		)

		finished := false // flag
		for {
			var (
				first *Message
				_out  chan *Message
			)

			if len(pending) > 0 {
				first = pending[0]
				_out = bf.out
			} else if finished {
				return
			}

			select {
			case incoming, ok := <-_in:
				if ok {
					pending = append(pending, incoming)
				} else {
					finished = true
					_in = nil
				}
			case _out <- first:
				pending = pending[1:]
			case <-bf.stop:
				close(bf.out) // Notify receiver
				return
			}
		}
	}()
}

func (bf *asyncBuff) Stop() (res bool) {
	close(bf.stop)
	bf.wg.Wait()
	return
}

// Biderectional socket
type asyncRWSocket struct {
	sync.Mutex
	conn          io.ReadWriteCloser
	upstreamBuf   *asyncBuff
	downstreamBuf *asyncBuff
	closed        chan struct{} // broadcast channel
}

func NewAsyncRW(conn io.ReadWriteCloser) (*asyncRWSocket, error) {
	sock := &asyncRWSocket{
		conn:          conn,
		upstreamBuf:   newAsyncBuf(),
		downstreamBuf: newAsyncBuf(),
		closed:        make(chan struct{}),
	}

	sock.readloop()
	sock.writeloop()

	return sock, nil
}

func NewUnixConnection(address string, timeout time.Duration) (SocketIO, error) {
	return newAsyncConnection("unix", address, timeout)
}

func NewTCPConnection(address string, timeout time.Duration) (SocketIO, error) {
	return newAsyncConnection("tcp", address, timeout)
}

func newAsyncConnection(family string, address string, timeout time.Duration) (SocketIO, error) {
	dialer := net.Dialer{
		Timeout:   timeout,
		DualStack: true,
	}

	conn, err := dialer.Dial(family, address)
	if err != nil {
		return nil, err
	}
	return NewAsyncRW(conn)
}

func (sock *asyncRWSocket) Close() {
	sock.upstreamBuf.Stop()
	sock.downstreamBuf.Stop()
	sock.close()
}

func (sock *asyncRWSocket) close() {
	sock.Lock()
	defer sock.Unlock()

	select {
	case <-sock.closed: // Already closed
	default:
		close(sock.closed)
		sock.conn.Close()
	}
}

func (sock *asyncRWSocket) IsClosed() (broadcast <-chan struct{}) {
	return sock.closed
}

func (sock *asyncRWSocket) Write() chan *Message {
	return sock.upstreamBuf.in
}

func (sock *asyncRWSocket) Read() chan *Message {
	return sock.downstreamBuf.out
}

func (sock *asyncRWSocket) writeloop() {
	go func() {
		encoder := codec.NewEncoder(sock.conn, hAsocket)
		for incoming := range sock.upstreamBuf.out {
			err := encoder.Encode(incoming)
			if err != nil {
				sock.close()
				return
			}
		}
	}()
}

func (sock *asyncRWSocket) readloop() {
	go func() {
		decoder := codec.NewDecoder(sock.conn, hAsocket)
		for {
			var message *Message
			err := decoder.Decode(&message)
			if err != nil {
				close(sock.downstreamBuf.in)
				sock.close()
				return
			}
			sock.downstreamBuf.in <- message
		}
	}()
}
