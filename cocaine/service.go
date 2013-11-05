package cocaine

import (
	"github.com/ugorji/go/codec"
	"log"
	"time"
)

type ServiceResult interface {
	Extract(interface{}) error
	Err() error
}

type ServiceRes struct {
	res []byte
	err error
}

func (s *ServiceRes) Extract(target interface{}) (err error) {
	err = codec.NewDecoderBytes(s.res, h).Decode(&target)
	return
}

func (s *ServiceRes) Err() error {
	return s.err
}

//
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
	finished := false
	go func() {
		var pending []ServiceResult
		for {
			var out chan ServiceResult
			var first ServiceResult
			if len(pending) > 0 {
				first = pending[0]
				out = Out
			} else if finished {
				close(Out)
				break
			}
			select {
			case incoming, ok := <-In:
				if ok {
					pending = append(pending, incoming)
				} else {
					finished = true
					In = nil
				}
			case out <- first:
				pending = pending[1:]
			}
		}
	}()
	return
}

type Service struct {
	sessions *Keeper
	unpacker *StreamUnpacker
	ResolveResult
	socketIO
}

func NewService(name string, args ...interface{}) *Service {
	l, err := NewLocator(args...)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer l.Close()
	info := <-l.Resolve(name)
	sock, err := NewASocket("tcp", info.Endpoint.AsString(), time.Second*5)
	if err != nil {
		log.Fatal(err)
	}
	s := Service{
		sessions:      NewKeeper(),
		unpacker:      NewStreamUnpacker(),
		ResolveResult: info,
		socketIO:      sock,
	}
	go s.loop()
	return &s
}

func (service *Service) loop() {
	for data := range service.socketIO.Read() {
		for _, item := range service.unpacker.Feed(data) {
			switch msg := item.(type) {
			case *Chunk:
				service.sessions.Get(msg.GetSessionID()) <- &ServiceRes{msg.Data, nil}
			case *Choke:
				close(service.sessions.Get(msg.GetSessionID()))
				service.sessions.Detach(msg.GetSessionID())
			case *ErrorMsg:
				service.sessions.Get(msg.GetSessionID()) <- &ServiceRes{nil, &ServiceError{msg.Code, msg.Message}}
			}
		}
	}
}

func (service *Service) Call(method int64, args ...interface{}) chan ServiceResult {
	in, out := GetServiceChanPair()
	id := service.sessions.Attach(in)
	msg := ServiceMethod{MessageInfo{method, id}, args}
	service.socketIO.Write() <- Pack(&msg)
	return out
}

func (service *Service) Close() {
	service.socketIO.Close()
}
