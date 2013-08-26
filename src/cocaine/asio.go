package cocaine

import (
	"log"
	"net"
)

func Transmite() (In chan RawMessage, Out chan RawMessage) {
	In = make(chan RawMessage)
	Out = make(chan RawMessage)
	go func() {
		log.Println("Start buffer")
		var pending []RawMessage
		for {
			var out chan RawMessage
			var first RawMessage
			if len(pending) > 0 {
				first = pending[0]
				out = Out
			}
			select {
			case incoming := <-In:
				pending = append(pending, incoming)
			case out <- first:
				pending = pending[1:]
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
		log.Println("Connection error", err)
	}
	pipe := Pipe{&conn, input, output}
	pipe.writeloop()
	pipe.readloop()
	return &pipe
}

func (pipe *Pipe) writeloop() {
	go func() {
		for incoming := range *pipe.input {
			log.Println("Write", incoming, len(incoming))
			count, err := (*pipe.conn).Write(incoming)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("Count", count)
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
				log.Println("Read error:", err)
			} else {
				log.Println("Receive", count, " bytes", buf[:count])
				*pipe.output <- buf[:count]
			}
		}
	}()
}
