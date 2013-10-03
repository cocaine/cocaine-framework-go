package cocaine

import (
	"github.com/ugorji/go/codec"
	"fmt"
	"log"
)

type ServiceResult struct {
	Res interface{}
	Err error
}

type ServiceError struct {
	Code    int
	Message string
}

func (err *ServiceError) Error() string {
	return err.Message
}

func GetServiceChanPair() (In chan ServiceResult, Out chan ServiceResult) {
	In = make(chan ServiceResult)
	Out = make(chan ServiceResult)
	go func() {
		var pending []ServiceResult
		for {
			var out chan ServiceResult
			var first ServiceResult
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

type Service struct {
	host     string
	port     uint64
	pipe     *Pipe
	unpacker *StreamUnpacker
	wr_in    chan RawMessage
	r_out    chan RawMessage
	sessions *Keeper
}

func (service *Service) loop() {
	for {
		for _, item := range service.unpacker.Feed(<-service.r_out) {
			switch msg := item.(type) {
			case *Chunk:
				//fmt.Println("Chunk", msg.GetSessionID(), msg.Data)
				var v interface{}
				err := codec.NewDecoderBytes(msg.Data, h).Decode(&v)
				fmt.Println(err)
				service.sessions.Get(msg.GetSessionID()) <- ServiceResult{v, nil}
			case *Choke:
				service.sessions.Get(msg.GetSessionID()) <- ServiceResult{nil, nil}
				service.sessions.Detach(msg.GetSessionID())
			case *ErrorMsg:
				fmt.Println("Error")
				service.sessions.Get(msg.GetSessionID()) <- ServiceResult{nil, &ServiceError{msg.Code, msg.Message}}
			}
		}
	}
}

func NewService(host string, port uint64, name string) *Service {
	log.Println("Create ", name)
	l := NewLocator(host, port)
	info := <-l.Resolve(name)
	wr_in, wr_out := Transmite()
	r_in, r_out := Transmite()
	fmt.Println(info.Endpoint.AsString())
	pipe := NewPipe("tcp", info.Endpoint.AsString(), &wr_out, &r_in)
	s := Service{host: info.Endpoint.Host,
		port:     info.Endpoint.Port,
		pipe:     pipe,
		unpacker: NewStreamUnpacker(),
		wr_in:    wr_in,
		r_out:    r_out,
		sessions: NewKeeper()}
	go s.loop()
	return &s
}

func (service *Service) Call(method int64, args ...interface{}) chan ServiceResult {
	in, out := GetServiceChanPair()
	id := service.sessions.Attach(in)
	msg := ServiceMethod{MessageInfo{method, id}, args}
	service.wr_in <- Pack(&msg)
	return out
}
