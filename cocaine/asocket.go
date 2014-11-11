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
	Read() chan GeneralMessage
	Write() chan GeneralMessage
	IsClosed() <-chan struct{}
	Close()
}

type asyncBuff struct {
	in, out chan GeneralMessage
	stop    chan bool
	wg      sync.WaitGroup
}

func newAsyncBuf() *asyncBuff {
	buf := asyncBuff{make(chan GeneralMessage), make(chan GeneralMessage), make(chan bool, 1), sync.WaitGroup{}}
	buf.loop()
	return &buf
}

func (bf *asyncBuff) loop() {
	go func() {
		bf.wg.Add(1)
		defer bf.wg.Done()
		var pending []GeneralMessage // data buffer
		var _in chan GeneralMessage  // incoming channel
		_in = bf.in

		finished := false // flag
		for {
			var first GeneralMessage
			var _out chan GeneralMessage
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
	conn           io.ReadWriteCloser
	clientToSock   *asyncBuff
	socketToClient *asyncBuff
	closed         chan struct{} //broadcast channel
	mutex          sync.Mutex
}

func newAsyncRWSocket(family string, address string, timeout time.Duration) (*asyncRWSocket, error) {
	conn, err := net.DialTimeout(family, address, timeout)
	if err != nil {
		return nil, err
	}

	sock := asyncRWSocket{conn, newAsyncBuf(), newAsyncBuf(), make(chan struct{}), sync.Mutex{}}
	sock.readloop()
	sock.writeloop()
	return &sock, nil
}

func (sock *asyncRWSocket) Close() {
	sock.clientToSock.Stop()
	sock.socketToClient.Stop()
	sock.close()
}

func (sock *asyncRWSocket) close() {
	sock.mutex.Lock()
	defer sock.mutex.Unlock()
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

func (sock *asyncRWSocket) Write() chan GeneralMessage {
	return sock.clientToSock.in
}

func (sock *asyncRWSocket) Read() chan GeneralMessage {
	return sock.socketToClient.out
}

func (sock *asyncRWSocket) writeloop() {
	go func() {
		h.StructToArray = true
		encoder := codec.NewEncoder(sock.conn, h)
		for incoming := range sock.clientToSock.out {
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
			var message GeneralMessage
			err := decoder.Decode(&message)
			if err != nil {
				close(sock.socketToClient.in)
				sock.close()
				return
			}
			sock.socketToClient.in <- message
		}
	}()
}
