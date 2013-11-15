package cocaine

import (
	"net"
	"time"
)

type socketIO interface {
	Read() chan RawMessage
	Write() chan RawMessage
	Close()
}

type socketWriter interface {
	Write() chan RawMessage
	Close()
}

type asyncBuff struct {
	in, out chan RawMessage
	stop    chan bool
}

func newAsyncBuf() *asyncBuff {
	buf := asyncBuff{make(chan RawMessage), make(chan RawMessage), make(chan bool, 1)}
	buf.loop()
	return &buf
}

func (bf *asyncBuff) loop() {
	go func() {
		var pending []RawMessage // data buffer
		var _in chan RawMessage  // incoming channel
		_in = bf.in

		finished := false // flag
		for {
			var first RawMessage
			var _out chan RawMessage
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
	return
}

// Biderectional socket
type ASocket struct {
	net.Conn
	closed         chan bool
	clientToSock   *asyncBuff
	socketToClient *asyncBuff
}

func NewASocket(family string, address string, timeout time.Duration) (*ASocket, error) {
	conn, err := net.DialTimeout(family, address, timeout)
	if err != nil {
		return nil, err
	}

	sock := ASocket{conn, make(chan bool), newAsyncBuf(), newAsyncBuf()}
	sock.readloop()
	sock.writeloop()
	return &sock, nil
}

func (sock *ASocket) Close() {
	close(sock.closed)
	sock.clientToSock.Stop()
	sock.socketToClient.Stop()
	sock.Conn.Close()
}

func (sock *ASocket) Write() chan RawMessage {
	return sock.clientToSock.in
}

func (sock *ASocket) Read() chan RawMessage {
	return sock.socketToClient.out
}

func (sock *ASocket) writeloop() {
	go func() {
		for incoming := range sock.clientToSock.out {
			_, err := sock.Conn.Write(incoming) //Add check for sending full
			if err != nil {
				return
			}
		}
	}()
}

func (sock *ASocket) readloop() {
	go func() {
		buf := make([]byte, 2048)
		for {
			count, err := sock.Conn.Read(buf)
			if err != nil {
				close(sock.socketToClient.in)
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
type WSocket struct {
	net.Conn
	state        chan bool
	clientToSock *asyncBuff
}

func NewWSocket(family string, address string, timeout time.Duration) (*WSocket, error) {
	conn, err := net.DialTimeout(family, address, timeout)
	if err != nil {
		return nil, err
	}

	sock := WSocket{conn, make(chan bool), newAsyncBuf()}
	sock.readloop()
	sock.writeloop()
	return &sock, nil
}

func (sock *WSocket) Write() chan RawMessage {
	return sock.clientToSock.in
}

func (sock *WSocket) Close() {
	sock.Conn.Close()
	sock.clientToSock.Stop()
}

func (sock *WSocket) writeloop() {
	go func() {
		for incoming := range sock.clientToSock.out {
			_, err := sock.Conn.Write(incoming) //Add check for sending full
			if err != nil {
				return
			}
		}
	}()
}

func (sock *WSocket) readloop() {
	go func() {
		buf := make([]byte, 2048)
		for {
			_, err := sock.Conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()
}
