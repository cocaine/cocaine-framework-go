package cocaine

import (
	"net"
	"sync"
	"time"

	"log"
)

var _ = log.Println

type socketIO interface {
	Read() chan rawMessage
	Write() chan rawMessage
	IsClosed() <-chan struct{}
	Close()
}

type socketWriter interface {
	Write() chan rawMessage
	IsClosed() <-chan struct{}
	Close()
}

type asyncBuff struct {
	in, out chan rawMessage
	stop    chan bool
	wg      sync.WaitGroup
}

func newAsyncBuf() *asyncBuff {
	buf := asyncBuff{make(chan rawMessage), make(chan rawMessage), make(chan bool, 1), sync.WaitGroup{}}
	buf.loop()
	return &buf
}

func (bf *asyncBuff) loop() {
	go func() {
		bf.wg.Add(1)
		defer bf.wg.Done()
		var pending []rawMessage // data buffer
		var _in chan rawMessage  // incoming channel
		_in = bf.in

		finished := false // flag
		for {
			var first rawMessage
			var _out chan rawMessage
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
				pending[0] = nil
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
	net.Conn
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
		sock.Conn.Close()
	}
}

func (sock *asyncRWSocket) IsClosed() (broadcast <-chan struct{}) {
	return sock.closed
}

func (sock *asyncRWSocket) Write() chan rawMessage {
	return sock.clientToSock.in
}

func (sock *asyncRWSocket) Read() chan rawMessage {
	return sock.socketToClient.out
}

func (sock *asyncRWSocket) writeloop() {
	go func() {
		for incoming := range sock.clientToSock.out {
			_, err := sock.Conn.Write(incoming) //Add check for sending full
			if err != nil {
				sock.close()
				return
			}
		}
	}()
}

func (sock *asyncRWSocket) readloop() {
	go func() {
		buf := make([]byte, 2048)
		for {
			count, err := sock.Conn.Read(buf)
			if err != nil {
				close(sock.socketToClient.in)
				sock.close()
				return
			} else {
				bufferToSend := make([]byte, count)
				copy(bufferToSend[:], buf[:count])
				sock.socketToClient.in <- bufferToSend
			}
		}
	}()
}

// WriteOnly Socket
type asyncWSocket struct {
	net.Conn
	clientToSock *asyncBuff
	closed       chan struct{} //broadcast channel
	mutex        sync.Mutex
}

func newWSocket(family string, address string, timeout time.Duration) (*asyncWSocket, error) {
	conn, err := net.DialTimeout(family, address, timeout)
	if err != nil {
		return nil, err
	}

	sock := asyncWSocket{conn, newAsyncBuf(), make(chan struct{}), sync.Mutex{}}
	sock.readloop()
	sock.writeloop()
	return &sock, nil
}

func (sock *asyncWSocket) Write() chan rawMessage {
	return sock.clientToSock.in
}

func (sock *asyncWSocket) close() {
	sock.mutex.Lock() // Is it really necessary???
	defer sock.mutex.Unlock()
	select {
	case <-sock.closed: // Already closed
	default:
		close(sock.closed)
		sock.Conn.Close()
	}
}

func (sock *asyncWSocket) Close() {
	sock.close()
	sock.clientToSock.Stop()
}

func (sock *asyncWSocket) IsClosed() (broadcast <-chan struct{}) {
	return sock.closed
}

func (sock *asyncWSocket) writeloop() {
	go func() {
		for incoming := range sock.clientToSock.out {
			_, err := sock.Conn.Write(incoming) //Add check for sending full
			if err != nil {
				sock.close()
				return
			}
		}
	}()
}

func (sock *asyncWSocket) readloop() {
	go func() {
		buf := make([]byte, 2048)
		for {
			_, err := sock.Conn.Read(buf)
			if err != nil {
				sock.close()
				return
			}
		}
	}()
}
