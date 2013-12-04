package cocaine

import (
	"time"

	"github.com/ugorji/go/codec"
)

type ServiceResult interface {
	Extract(interface{}) error
	Err() error
}

type serviceRes struct {
	res []byte
	err error
}

func (s *serviceRes) Extract(target interface{}) (err error) {
	err = codec.NewDecoderBytes(s.res, h).Decode(&target)
	return
}

func (s *serviceRes) Err() error {
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

func getServiceChanPair(stop <-chan bool) (In chan ServiceResult, Out chan ServiceResult) {
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

			case <-stop: // Notification from Close()
				return
			}
		}
	}()
	return
}

type Service struct {
	sessions *keeperStruct
	unpacker *streamUnpacker
	stop     chan bool
	ResolveResult
	socketIO
}

func NewService(name string, args ...interface{}) (s *Service, err error) {
	l, err := NewLocator(args...)
	if err != nil {
		return
	}
	defer l.Close()
	info := <-l.Resolve(name)
	sock, err := newAsyncRWSocket("tcp", info.Endpoint.AsString(), time.Second*5)
	if err != nil {
		return
	}
	s = &Service{
		sessions:      newKeeperStruct(),
		unpacker:      newStreamUnpacker(),
		stop:          make(chan bool),
		ResolveResult: info,
		socketIO:      sock,
	}
	go s.loop()
	return
}

func (service *Service) loop() {
	for data := range service.socketIO.Read() {
		for _, item := range service.unpacker.Feed(data) {
			switch msg := item.(type) {
			case *chunk:
				if ch, ok := service.sessions.Get(msg.getSessionID()); ok {
					ch <- &serviceRes{msg.Data, nil}
				}
			case *choke:
				if ch, ok := service.sessions.Get(msg.getSessionID()); ok {
					close(ch)
					service.sessions.Detach(msg.getSessionID())
				}
			case *errorMsg:
				if ch, ok := service.sessions.Get(msg.getSessionID()); ok {
					ch <- &serviceRes{nil, &ServiceError{msg.Code, msg.Message}}
				}
			}
		}
	}
}

func (service *Service) Call(name string, args ...interface{}) chan ServiceResult {
	method, err := service.getMethodNumber(name)
	if err != nil {
		errorOut := make(chan ServiceResult, 1)
		errorOut <- &serviceRes{nil, &ServiceError{-100, "Wrong method name"}}
		return errorOut
	}
	in, out := getServiceChanPair(service.stop)
	id := service.sessions.Attach(in)
	msg := ServiceMethod{messageInfo{method, id}, args}
	service.socketIO.Write() <- packMsg(&msg)
	return out
}

func (service *Service) Close() {
	close(service.stop) // Broadcast all related goroutines about disposing
	service.socketIO.Close()
}
