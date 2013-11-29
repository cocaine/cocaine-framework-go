package cocaine

import (
	"sync"
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

func getServiceChanPair(stop <-chan int) (In chan ServiceResult, Out chan ServiceResult) {
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

/*
Service's Finite-state Machine

DISCONNECTED <-> CONNECTING -> CONNECTED

CONNECTED -> DISCONNECTED

*/

type serviceFSM struct {
	DISCONNECTED, CONNECTING, CONNECTED, CLOSED chan int
	_TRUE                                       chan int // represents as close channel
	_FALSE                                      chan int // represents as nil channel
	sync.Mutex
}

func (fsm *serviceFSM) setConnected() {
	fsm.Lock()
	defer fsm.Unlock()
	select {
	case <-fsm.CONNECTED:
		return
	case <-fsm.CLOSED:
		return
	default:
		fsm.DISCONNECTED, fsm.CONNECTING = fsm._FALSE, fsm._FALSE
		fsm.CONNECTED = fsm._TRUE
	}
}

func (fsm *serviceFSM) setDisconnected() {
	fsm.Lock()
	defer fsm.Unlock()
	select {
	case <-fsm.DISCONNECTED:
		return
	default:
		fsm.CONNECTED, fsm.CONNECTING = fsm._FALSE, fsm._FALSE
		fsm.DISCONNECTED = fsm._TRUE
	}
}

func (fsm *serviceFSM) setConnecting() {
	fsm.Lock()
	defer fsm.Unlock()
	select {
	case <-fsm.CONNECTING:
		return
	default:
		fsm.CONNECTED, fsm.DISCONNECTED = fsm._FALSE, fsm._FALSE
		fsm.CONNECTING = fsm._TRUE
	}
}

func (fsm *serviceFSM) setClosed() {
	fsm.Lock()
	defer fsm.Unlock()
	select {
	case <-fsm.CLOSED:
		return
	default:
		fsm.CONNECTED, fsm.CONNECTING, fsm.DISCONNECTED = fsm._FALSE, fsm._FALSE, fsm._FALSE
		fsm.CLOSED = fsm._TRUE
	}
}

type ServiceInfo struct {
	name string
	args []interface{}
}

func (s *ServiceInfo) Name() string {
	return s.name
}

func (s *ServiceInfo) ConnectionArgs() []interface{} {
	return s.args
}

type Service struct {
	sessions *keeperStruct
	unpacker *streamUnpacker
	mutex    sync.Mutex
	serviceFSM
	ResolveResult
	socketIO
	ServiceInfo
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
	state := make(chan int)
	close(state)
	fsm := serviceFSM{_TRUE: state, CONNECTED: state}
	s = &Service{
		sessions:      newKeeperStruct(),
		unpacker:      newStreamUnpacker(),
		mutex:         sync.Mutex{},
		serviceFSM:    fsm,
		ResolveResult: info,
		socketIO:      sock,
		ServiceInfo:   ServiceInfo{name, args},
	}
	go s.loop()
	return
}

func (service *Service) loop() {
	for {
		select {
		case data, ok := <-service.socketIO.Read():
			if !ok {
				return
			}
			for _, item := range service.unpacker.Feed(data) {
				switch msg := item.(type) {
				case *chunk:
					if ch, ok := service.sessions.Get(msg.getSessionID()); ok {
						ch <- &serviceRes{msg.Data, nil}
					}
				case *choke:
					if ch, ok := service.sessions.Get(msg.getSessionID()); ok {
						service.sessions.Detach(msg.getSessionID())
						close(ch)
					}
				case *errorMsg:
					if ch, ok := service.sessions.Get(msg.getSessionID()); ok {
						ch <- &serviceRes{nil, &ServiceError{msg.Code, msg.Message}}
					}
				}
			}
		case <-service.CLOSED:
			return
		case <-service.DISCONNECTED:
			return
		}
	}
}

func (service *Service) Reconnect(args ...interface{}) (err error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	select {
	case <-service.CONNECTING:
		return
	default:
		// Drop all old sessions. Send error to channels.
		for _, key := range service.sessions.Keys() {
			if ch, ok := service.sessions.Get(key); ok {
				ch <- &serviceRes{nil, &ServiceError{-32, "Connection has broken"}}
				service.sessions.Detach(key)
				close(ch)
			}
		}
		var swapCandidate *Service
		if len(args) > 0 {
			swapCandidate, err = NewService(service.Name(), args...)
		} else {
			swapCandidate, err = NewService(service.Name(), service.ConnectionArgs()...)
		}
		if err != nil {
			service.serviceFSM.setDisconnected()
			return
		}
		service.Close()
		service = swapCandidate

	}
	return
}

func (service *Service) Call(name string, args ...interface{}) chan ServiceResult {
	method, err := service.getMethodNumber(name)
	if err != nil {
		errorOut := make(chan ServiceResult, 1)
		errorOut <- &serviceRes{nil, &ServiceError{-100, "Wrong method name"}}
		return errorOut
	}
	in, out := getServiceChanPair(service.serviceFSM.CLOSED)
	id := service.sessions.Attach(in)
	msg := ServiceMethod{messageInfo{method, id}, args}
	service.socketIO.Write() <- packMsg(&msg)
	return out
}

func (service *Service) Close() {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	select {
	case <-service.CLOSED: // Service has been already closed
	default:
		service.serviceFSM.setClosed() // Broadcast all related goroutines about disposing
		service.socketIO.Close()
	}
}
