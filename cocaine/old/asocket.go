package cocaine

import (
	"io"
	"net"
	"sync"
	"time"

	"log"

	"github.com/ugorji/go/codec"
)

var (
	mh codec.MsgpackHandle
	h  = &mh
)

var _ = log.Println

type socketIO interface {
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
	buf := asyncBuff{
		in:   make(chan *Message),
		out:  make(chan *Message),
		stop: make(chan struct{})}

	buf.loop()
	return &buf
}

func (bf *asyncBuff) loop() {
	go func() {
		bf.wg.Add(1)
		defer bf.wg.Done()
		var pending []*Message // data buffer
		var _in chan *Message  // incoming channel
		_in = bf.in

		finished := false // flag
		for {
			var first *Message
			var _out chan *Message
			if len(pending) > 0 {
				first = pending[0]
				_out = bf.out
			} else if finished {
				break
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
	closed        chan struct{} //broadcast channel
}

func newAsyncRW(conn io.ReadWriteCloser) (*asyncRWSocket, error) {
	sock := asyncRWSocket{
		conn:          conn,
		upstreamBuf:   newAsyncBuf(),
		downstreamBuf: newAsyncBuf(),
		closed:        make(chan struct{}),
	}
	sock.readloop()
	sock.writeloop()
	return &sock, nil
}

func newAsyncConnection(family string, address string, timeout time.Duration) (*asyncRWSocket, error) {
	conn, err := net.DialTimeout(family, address, timeout)
	if err != nil {
		return nil, err
	}
	return newAsyncRW(conn)
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
		h.StructToArray = true
		encoder := codec.NewEncoder(sock.conn, h)
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
		decoder := codec.NewDecoder(sock.conn, h)
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
