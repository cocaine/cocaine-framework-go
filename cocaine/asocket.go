package cocaine

import (
	"net"
	"sync"
	"time"

	log "gitlab.srv.pv.km/common/logger"
)

//var _ = log.Println

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
	log.Info("Create new asyncBuff")
	buf := asyncBuff{make(chan rawMessage), make(chan rawMessage), make(chan bool, 1), sync.WaitGroup{}}
	buf.loop()
	return &buf
}

func (bf *asyncBuff) loop() {
	go func() {
		log.Info("[asyncBuff] loop(): start")
		bf.wg.Add(1)
		defer bf.wg.Done()
		var pending []rawMessage // data buffer
		var _in chan rawMessage  // incoming channel
		_in = bf.in

		finished := false // flag
		for {
			log.Info("[asyncBuff] loop(): next iteration")
			var first rawMessage
			var _out chan rawMessage
			if len(pending) > 0 {
				log.Info("[asyncBuff] loop(): changing the value of the first element")
				first = pending[0]
				_out = bf.out
			} else if finished {
				log.Info("[asyncBuff] loop(): finished")
				close(bf.out)
				log.Error("[asyncBuff] loop(): output channel is closed")
				break
			}
			select {
			case incoming, ok := <-_in:
				if ok {
					log.Info("[asyncBuff] loop(): got rawMessage")
					pending = append(pending, incoming)
				} else {
					log.Error("[asyncBuff] loop(): input channel is closed")
					finished = true
					_in = nil
				}
			case _out <- first:
				log.Infof("[asyncBuff] loop(): first element has been sent to channel")
				pending[0] = nil
				pending = pending[1:]
			case <-bf.stop:
				log.Infof("[asyncBuff] loop(): Notification from bf.stop")
				close(bf.out) // Notify receiver
				log.Info("[asyncBuff] loop(): end")
				return
			}
		}
		log.Info("[asyncBuff] loop(): end")
	}()
}

func (bf *asyncBuff) Stop() (res bool) {
	log.Info("[asyncBuff] Stop(): start")
	close(bf.stop)
	bf.wg.Wait()
	log.Info("[asyncBuff] Stop(): end")
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
	dialer := net.Dialer{
		Timeout:   timeout,
		DualStack: true,
	}

	conn, err := dialer.Dial(family, address)
	if err != nil {
		return nil, err
	}

	sock := asyncRWSocket{conn, newAsyncBuf(), newAsyncBuf(), make(chan struct{}), sync.Mutex{}}
	sock.readloop()
	sock.writeloop()
	return &sock, nil
}

func (sock *asyncRWSocket) Close() {
	log.Error("[asyncRWSocket] Close() start")
	sock.clientToSock.Stop()
	sock.socketToClient.Stop()
	sock.close()
	log.Error("[asyncRWSocket] Close() end")
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
		log.Infof("[asyncRWSocket] writeloop() start")
		for incoming := range sock.clientToSock.out {
			log.Infof("[asyncRWSocket] writeloop() write data")
			_, err := sock.Conn.Write(incoming) //Add check for sending full
			if err != nil {
				log.Errorf("[asyncRWSocket] writeloop() error: %v", err)
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
			log.Infof("[asyncRWSocket] writeloop() data has been written")
		}
		log.Infof("[asyncRWSocket] writeloop() end")
	}()
}

func (sock *asyncRWSocket) readloop() {
	go func() {
		buf := make([]byte, 65536)
		log.Infof("[asyncRWSocket] readloop() start")
		for {
			count, err := sock.Conn.Read(buf)
			log.Infof("[asyncRWSocket] readloop() got data")
			if count > 0 {
				bufferToSend := make([]byte, count)
				copy(bufferToSend[:], buf[:count])
				log.Infof("[asyncRWSocket] readloop() send data to channel")
				sock.socketToClient.in <- bufferToSend
				log.Info("[asyncRWSocket] readloop(): data has been sent to channel")
			}

			if err != nil {
				log.Errorf("[asyncRWSocket] readloop() error: %v", err)
				close(sock.socketToClient.in)
				sock.close()
				return
			}
		}
		log.Infof("[asyncRWSocket] readloop() end")
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
