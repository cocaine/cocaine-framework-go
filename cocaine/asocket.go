package cocaine

import (
	"net"
	"sync"
	"time"

	"log"
)

var _ = log.Println

type LocalLogger interface {
	Debug(args ...interface{})
	Debugf(msg string, args ...interface{})
	Info(args ...interface{})
	Infof(msg string, args ...interface{})
	Warn(args ...interface{})
	Warnf(msg string, args ...interface{})
	Err(args ...interface{})
	Errf(msg string, args ...interface{})
}

type LocalLoggerImpl struct{}

func (l *LocalLoggerImpl) Debug(args ...interface{})              { /*pass*/ }
func (l *LocalLoggerImpl) Debugf(msg string, args ...interface{}) { /*pass*/ }
func (l *LocalLoggerImpl) Info(args ...interface{})               { log.Print(args) }
func (l *LocalLoggerImpl) Infof(msg string, args ...interface{})  { log.Printf(msg, args) }
func (l *LocalLoggerImpl) Warn(args ...interface{})               { log.Print(args) }
func (l *LocalLoggerImpl) Warnf(msg string, args ...interface{})  { log.Printf(msg, args) }
func (l *LocalLoggerImpl) Err(args ...interface{})                { log.Print(args) }
func (l *LocalLoggerImpl) Errf(msg string, args ...interface{})   { log.Printf(msg, args) }

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
				close(bf.out)
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
	logger         LocalLogger
}

func newAsyncRWSocket(family string, address string, timeout time.Duration, logger LocalLogger) (*asyncRWSocket, error) {
	dialer := net.Dialer{
		Timeout:   timeout,
		DualStack: true,
	}

	conn, err := dialer.Dial(family, address)
	if err != nil {
		return nil, err
	}

	sock := asyncRWSocket{conn, newAsyncBuf(), newAsyncBuf(), make(chan struct{}), sync.Mutex{}, logger}
	sock.readloop()
	sock.writeloop()
	return &sock, nil
}

func (sock *asyncRWSocket) Close() {
	sock.clientToSock.Stop()
	sock.socketToClient.Stop()
	sock.close()
}

func (sock *asyncRWSocket) close() bool {
	sock.mutex.Lock()
	defer sock.mutex.Unlock()
	select {
	case <-sock.closed: // Already closed
	default:
		close(sock.closed)
		sock.Conn.Close()
		return true
	}
	return false
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
				sock.logger.Errf("Error while writing data: %v", err)
				sock.close()
				// blackhole all writes
				go func() {
					ok := true
					for ok {
						_, ok = <-sock.clientToSock.out
					}
				}()
				return
			}
		}
	}()
}

func (sock *asyncRWSocket) readloop() {
	go func() {
		buf := make([]byte, 65536)
		for {
			count, err := sock.Conn.Read(buf)
			if count > 0 {
				bufferToSend := make([]byte, count)
				copy(bufferToSend[:], buf[:count])
				sock.socketToClient.in <- bufferToSend
			}

			if err != nil {
				close(sock.socketToClient.in)
				if sock.close() {
					// pass errors like "use of closed network connection"
					sock.logger.Errf("Error while reading data: %v", err)
				}
				return
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
	logger       LocalLogger
}

func newWSocket(family string, address string, timeout time.Duration, logger LocalLogger) (*asyncWSocket, error) {
	conn, err := net.DialTimeout(family, address, timeout)
	if err != nil {
		return nil, err
	}

	sock := asyncWSocket{conn, newAsyncBuf(), make(chan struct{}), sync.Mutex{}, logger}
	sock.readloop()
	sock.writeloop()
	return &sock, nil
}

func (sock *asyncWSocket) Write() chan rawMessage {
	return sock.clientToSock.in
}

func (sock *asyncWSocket) close() bool {
	sock.mutex.Lock() // Is it really necessary???
	defer sock.mutex.Unlock()
	select {
	case <-sock.closed: // Already closed
	default:
		close(sock.closed)
		sock.Conn.Close()
		return true
	}
	return false
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
				sock.logger.Errf("Error while writing data: %v", err)
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
				if sock.close() {
					// pass errors like "use of closed network connection"
					sock.logger.Errf("Error while reading data: %v", err)
				}
				return
			}
		}
	}()
}
