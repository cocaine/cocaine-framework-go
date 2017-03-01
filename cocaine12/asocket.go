package cocaine12

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/tinylib/msgp/msgp"
)

// var (
// 	mhAsocket = codec.MsgpackHandle{
// 		BasicHandle: codec.BasicHandle{
// 			EncodeOptions: codec.EncodeOptions{
// 				StructToArray: true,
// 			},
// 		},
// 	}
// 	hAsocket = &mhAsocket
// )

type asyncSender interface {
	Send(*message)
}

type socketIO interface {
	asyncSender
	Read() chan *message
	Write() chan *message
	IsClosed() <-chan struct{}
	Close()
}

type asyncBuff struct {
	in  chan *message
	out chan *message

	stop chan (<-chan time.Time)
	wait chan struct{}
}

func newAsyncBuf() *asyncBuff {
	buf := &asyncBuff{
		in:  make(chan *message),
		out: make(chan *message),

		// to stop my loop
		stop: make(chan (<-chan time.Time)),
		// to wait for a notifycation
		// from the loop that it's stopped
		wait: make(chan struct{}),
	}

	buf.loop()
	return buf
}

func (bf *asyncBuff) loop() {
	go func() {
		defer close(bf.wait)

		// Notify a receiver
		defer close(bf.out)

		var (
			// buffer for messages
			pending []*message

			// it should be read until closed
			// to get all messages from a sender
			input = bf.in

			// if <-chan time.Time is received we have to wait the buffer drainig
			// if closed return immediatly
			stopped = bf.stop

			quitAfterTimeout <-chan time.Time

			// empty buffer & the input is closed
			finished = false
		)

		for {
			var (
				candidate *message
				out       chan *message
			)

			if len(pending) > 0 {
				// mark the first message as a candidate to be sent
				// and unlock the sending state
				candidate = pending[0]
				out = bf.out
			} else if finished {
				// message queue is empty and
				// no more messages are expected
				return
			}

			select {
			// get a message from a sender
			case incoming, open := <-input:
				if open {
					pending = append(pending, incoming)
				} else {
					// Set the flag
					// Unset channel to lock the case
					finished = true
					input = nil
				}

			// send the first message from the queue to a reveiver
			case out <- candidate:
				pending = pending[1:]

			case timeoutChan, open := <-stopped:
				if !open {
					return
				}

				// Disable this case
				// to protect from Stop() after Drain()
				stopped = nil
				quitAfterTimeout = timeoutChan

			// it's usually nil channel, but
			// it can be set using passing a chan time.Time via stopped
			case <-quitAfterTimeout:
				return
			}
		}
	}()
}

// Stop stops a loop which is handling messages in the buffer
// It is prohibited to call Drain afer Stop
func (bf *asyncBuff) Stop() error {
	close(bf.stop)
	select {
	case <-bf.wait:
	case <-time.After(time.Second):
	}

	return nil
}

// Drain waits for the duration to let the buffer send pending messages.
// It is prohibited to call Drain after Stop
func (bf *asyncBuff) Drain(d time.Duration) error {
	var timeoutChan = time.After(d)
	select {
	case bf.stop <- timeoutChan:
	case <-timeoutChan:
	}
	return bf.Stop()
}

// Biderectional socket
type asyncRWSocket struct {
	sync.Mutex
	conn          io.ReadWriteCloser
	upstreamBuf   *asyncBuff
	downstreamBuf *asyncBuff
	closed        chan struct{} // broadcast channel
}

func newAsyncRW(conn io.ReadWriteCloser) (*asyncRWSocket, error) {
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

func newUnixConnection(address string, timeout time.Duration) (socketIO, error) {
	return newAsyncConnection("unix", address, timeout)
}

func newTCPConnection(address string, timeout time.Duration) (socketIO, error) {
	return newAsyncConnection("tcp", address, timeout)
}

func newAsyncConnection(family string, address string, timeout time.Duration) (socketIO, error) {
	dialer := net.Dialer{
		Timeout:   timeout,
		DualStack: true,
	}

	conn, err := dialer.Dial(family, address)
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

func (sock *asyncRWSocket) Write() chan *message {
	return sock.upstreamBuf.in
}

func (sock *asyncRWSocket) Read() chan *message {
	return sock.downstreamBuf.out
}

func (sock *asyncRWSocket) Send(msg *message) {
	select {
	case sock.Write() <- msg:
	case <-sock.IsClosed():
		// Socket is in the closed state,
		// so drop the data
	}
}

func (sock *asyncRWSocket) writeloop() {
	go func() {
		wr := msgp.NewWriter(sock.conn)
		for incoming := range sock.upstreamBuf.out {
			if err := incoming.EncodeMsg(wr); err != nil {
				sock.close()
				// blackhole all pending writes. See #31
				go func() {
					for _ = range sock.upstreamBuf.out {
						// pass
					}
				}()
				return
			}
			wr.Flush()
		}
	}()
}

func (sock *asyncRWSocket) readloop() {
	go func() {
		r := msgp.NewReader(sock.conn)
		for {
			var msg message
			if err := msg.DecodeMsg(r); err != nil {
				close(sock.downstreamBuf.in)
				sock.close()
				return
			}
			sock.downstreamBuf.in <- &msg
		}
	}()
}
