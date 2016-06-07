package cocaine

import (
	"fmt"
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

//Unpacks the result of the called method in the passed structure.
//You can transfer the structure of a particular type that will avoid the type checking. Look at examples.
func (s *serviceRes) Extract(target interface{}) (err error) {
	err = codec.NewDecoderBytes(s.res, h).Decode(&target)
	return
}

//Error status
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

//Allows you to invoke methods of services and send events to other cloud applications.
type Service struct {
	sessions *keeperStruct
	unpacker *streamUnpacker
	stop     chan bool
	args     []interface{}
	name     string
	ResolveResult
	socketIO
	mutex           sync.Mutex
	wg              sync.WaitGroup
	is_reconnecting bool
}

//Creates new service instance with specifed name.
//Optional parameter is a network endpoint of the locator (default ":10053"). Look at Locator.
func serviceResolve(name string, args ...interface{}) (info ResolveResult, err error) {
	l, err := NewLocator(args...)
	if err != nil {
		return
	}
	defer l.Close()
	info = <-l.Resolve(name)
	return
}

func serviceCreateIO(endpoint string) (sock socketIO, err error) {
	sock, err = newAsyncRWSocket("tcp", endpoint, time.Second*5)
	return
}

func NewService(name string, args ...interface{}) (s *Service, err error) {
	info, err := serviceResolve(name, args...)
	if err != nil {
		return
	}

	if !info.success {
		err = fmt.Errorf("Unable to resolve service %s", name)
		return
	}

	sock, err := serviceCreateIO(info.Endpoint.AsString())
	if err != nil {
		return
	}

	s = &Service{
		sessions:        newKeeperStruct(),
		unpacker:        newStreamUnpacker(),
		stop:            make(chan bool),
		args:            args,
		name:            name,
		ResolveResult:   info,
		socketIO:        sock,
		mutex:           sync.Mutex{},
		wg:              sync.WaitGroup{},
		is_reconnecting: false,
	}
	go s.loop()
	return
}

func (service *Service) loop() {
	for data := range service.socketIO.Read() {
		if data == nil {
			err := ServiceError{-1, "undefined error"}
			for _, id := range service.sessions.Keys() {
				if ch, ok := service.sessions.Get(id); ok {
					ch <- &serviceRes{nil, &err}
					close(ch)
					service.sessions.Detach(id)
				}
			}
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

func (service *Service) Reconnect(force bool) error {
	if !service.is_reconnecting {
		service.mutex.Lock()
		defer service.mutex.Unlock()
		if service.is_reconnecting {
			return fmt.Errorf("%s", "Service is reconnecting now")
		}
		service.is_reconnecting = true
		defer func() { service.is_reconnecting = false }()

		if !force {
			select {
			case <-service.IsClosed():
			default:
				return fmt.Errorf("%s", "Service is already connected")
			}
		}
		// Send error to all open sessions
		for _, key := range service.sessions.Keys() {
			service.sessions.RLock()
			fmt.Println(key)
			if ch, ok := service.sessions.Get(key); ok {
				ch <- &serviceRes{nil, &ServiceError{-100, "Disconnected"}}
			}
			service.sessions.RUnlock()
			service.sessions.Detach(key)
		}

		// Create new socket
		info, err := serviceResolve(service.name, service.args...)
		if err != nil {
			return err
		}
		sock, err := serviceCreateIO(info.Endpoint.AsString())
		if err != nil {
			return err
		}

		// Dispose old IO interface
		service.Close()
		service.stop = make(chan bool)
		service.socketIO = sock
		service.unpacker = newStreamUnpacker()
		go service.loop()
		return nil
	}
	return fmt.Errorf("%s", "Service is reconnecting now")
}

func (service *Service) call(name string, args ...interface{}) chan ServiceResult {
	method, err := service.getMethodNumber(name)
	if err != nil {
		errorOut := make(chan ServiceResult, 1)
		errorOut <- &serviceRes{nil, &ServiceError{-100, "Wrong method name"}}
		return errorOut
	}
	in, out := service.getServiceChanPair()
	id := service.sessions.Attach(in)
	msg := ServiceMethod{messageInfo{method, id}, args}
	service.socketIO.Write() <- packMsg(&msg)
	return out
}

//Calls a remote method by name and pass args
func (service *Service) Call(name string, args ...interface{}) chan ServiceResult {
	select {
	case <-service.IsClosed():
		if err := service.Reconnect(false); err != nil {
			errorOut := make(chan ServiceResult, 1)
			errorOut <- &serviceRes{nil, &ServiceError{-32, "Disconnected"}}
			return errorOut
		}
	default:
	}
	return service.call(name, args...)
}

//Disposes resources of a service. You must call this method if the service isn't used anymore.
func (service *Service) Close() {
	close(service.stop) // Broadcast all related goroutines about disposing
	service.socketIO.Close()
}

func (service *Service) getServiceChanPair() (In chan ServiceResult, Out chan ServiceResult) {
	In = make(chan ServiceResult)
	Out = make(chan ServiceResult)
	go func() {
		service.wg.Add(1)
		defer service.wg.Done()
		finished := false
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

			case <-service.stop: // Notification from Close()
				return
			}
		}
	}()
	return
}
