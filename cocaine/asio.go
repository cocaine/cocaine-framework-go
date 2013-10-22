package cocaine

import (
	"log"
	"net"
	"runtime/debug"
)

func Transmite() (In chan RawMessage, Out chan RawMessage) {
	In = make(chan RawMessage)
	Out = make(chan RawMessage)
	go func() {
		var pending []RawMessage
		for {
			var first RawMessage
			if len(pending) > 0 {
				first = pending[0]

				select {
				case incoming := <-In:
					pending = append(pending, incoming)
				case Out <- first:
					pending[0] = nil
					pending = pending[1:]
				}
			} else {
				incoming := <-In
				pending = append(pending, incoming)
			}
		}
	}()
	return
}

type Pipe struct {
	conn   *net.Conn
	input  *chan RawMessage
	output *chan RawMessage
}

func NewPipe(family, endpoint string, input *chan RawMessage, output *chan RawMessage) *Pipe {
	conn, err := net.Dial(family, endpoint)
	if err != nil {
		log.Fatal("Connection error", err, family, endpoint, string(debug.Stack()))
	}
	pipe := Pipe{&conn, input, output}
	pipe.writeloop()
	pipe.readloop()
	return &pipe
}

func (pipe *Pipe) writeloop() {
	go func() {
		for incoming := range *pipe.input {
			_, err := (*pipe.conn).Write(incoming)
			if err != nil {
				log.Fatal(err)
			}
		}
	}()
}

func (pipe *Pipe) readloop() {
	go func() {
		buf := make([]byte, 1024)
		for {
			count, err := (*pipe.conn).Read(buf)
			if err != nil {
				log.Fatal("Read error:", err)
			} else {
				bufferToSend := make([]byte, count)
				copy(bufferToSend[:], buf[:count])
				*pipe.output <- bufferToSend
			}
		}
	}()
}
