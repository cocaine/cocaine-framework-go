package cocaine

import (
	"fmt"
	"github.com/ugorji/go/codec"
	log "gitlab.srv.pv.km/common/logger"
	"sync"
	"time"
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

//func getServiceChanPair(stop <-chan bool) (In chan ServiceResult, Out chan ServiceResult) {
//	In = make(chan ServiceResult)
//	Out = make(chan ServiceResult)
//	finished := false
//	go func() {
//		var pending []ServiceResult
//		for {
//			var out chan ServiceResult
//			var first ServiceResult

//			if len(pending) > 0 {
//				first = pending[0]
//				out = Out
//			} else if finished {
//				close(Out)
//				break
//			}

//			select {
//			case incoming, ok := <-In:
//				if ok {
//					pending = append(pending, incoming)
//				} else {
//					finished = true
//					In = nil
//				}

//			case out <- first:
//				pending = pending[1:]

//			case <-stop: // Notification from Close()
//				return
//			}
//		}
//	}()
//	return
//}

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
	log.Infof("[%v] Create service", name)
	info, err := serviceResolve(name, args...)
	if err != nil {
		log.Errorf("[%v] Resolve error: %v", name, err)
		return
	}

	if !info.success {
		err = fmt.Errorf("Unable to resolve service %s", name)
		log.Errorf("[%v] %v", name, err)
		return
	}

	sock, err := serviceCreateIO(info.Endpoint.AsString())
	if err != nil {
		log.Errorf("[%v] Create error: %v", name, err)
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
	log.Infof("[%v] Service has been created", name)
	go s.loop()
	return
}

func (service *Service) loop() {
	log.Infof("[%v] loop(): start", service.name)

	for data := range service.socketIO.Read() {
		for _, item := range service.unpacker.Feed(data) {
			switch msg := item.(type) {
			case *chunk:
				log.Infof("[%v] loop(): got chunk", service.name)
				if ch, ok := service.sessions.Get(msg.getSessionID()); ok {
					log.Infof("[%v] loop(): send chunk to channel", service.name)
					ch <- &serviceRes{msg.Data, nil}
					log.Infof("[%v] loop(): chank has been sent to channel", service.name)
				}
			case *choke:
				log.Infof("[%v] loop(): got choke", service.name)
				if ch, ok := service.sessions.Get(msg.getSessionID()); ok {
					log.Infof("[%v] loop(): close the channel", service.name)
					close(ch)
					service.sessions.Detach(msg.getSessionID())
					log.Infof("[%v] loop(): Channel has been closed", service.name)
				}
			case *errorMsg:
				log.Infof("[%v] loop(): got errorMsg", service.name)
				if ch, ok := service.sessions.Get(msg.getSessionID()); ok {
					log.Infof("[%v] loop(): send errorMsg to channel", service.name)
					ch <- &serviceRes{nil, &ServiceError{msg.Code, msg.Message}}
					log.Infof("[%v] loop(): errorMsg has been sent to channel", service.name)
				}
			default:
				log.Errorf("[%v] loop(): got unknown message", service.name)
			}
		}
	}
	log.Infof("[%v] loop(): Send error to all open sessions", service.name)
	for _, id := range service.sessions.Keys() {
		if ch, ok := service.sessions.Get(id); ok {
			ch <- &serviceRes{nil, &ServiceError{-1, "Disconnected"}}
			close(ch)
			service.sessions.Detach(id)
		}
	}
	log.Infof("[%v] loop(): end", service.name)
}

func (service *Service) Reconnect(force bool) error {
	log.Infof("[%v] Reconnect(): start", service.name)
	if !service.is_reconnecting {
		service.mutex.Lock()
		defer service.mutex.Unlock()
		if service.is_reconnecting {
			log.Errorf("[%v] Reconnect(): Service is reconnecting now", service.name)
			return fmt.Errorf("%s", "Service is reconnecting now")
		}
		service.is_reconnecting = true
		defer func() { service.is_reconnecting = false }()

		if !force {
			select {
			case <-service.IsClosed():
			default:
				log.Errorf("[%v] Reconnect(): Service is already connected", service.name)
				return fmt.Errorf("%s", "Service is already connected")
			}
		}
		// Send error to all open sessions
		log.Infof("[%v] Reconnect(): Send error to all open sessions", service.name)
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
		log.Infof("[%v] Reconnect(): Create new socket", service.name)
		info, err := serviceResolve(service.name, service.args...)
		if err != nil {
			log.Errorf("[%v] Reconnect(): resolve error: %v", service.name, err)
			return err
		}
		sock, err := serviceCreateIO(info.Endpoint.AsString())
		if err != nil {
			log.Errorf("[%v] Reconnect(): create error: %v", service.name, err)
			return err
		}

		// Dispose old IO interface
		log.Infof("[%v] Reconnect(): Dispose old IO interface", service.name)
		service.Close()
		service.stop = make(chan bool)
		service.socketIO = sock
		service.unpacker = newStreamUnpacker()

		log.Infof("[%v] Reconnect(): run loop()", service.name)
		go service.loop()
		return nil
	}
	log.Errorf("[%v] Reconnect(): Service is reconnecting now", service.name)
	return fmt.Errorf("%s", "Service is reconnecting now")
}

func (service *Service) call(name string, args ...interface{}) chan ServiceResult {
	log.Infof("[%v] call(): start", service.name)
	method, err := service.getMethodNumber(name)
	if err != nil {
		log.Infof("[%v] call(): error: %v", service.name, err)
		errorOut := make(chan ServiceResult, 1)
		errorOut <- &serviceRes{nil, &ServiceError{-100, "Wrong method name"}}
		return errorOut
	}
	in, out := service.getServiceChanPair()
	id := service.sessions.Attach(in)
	msg := ServiceMethod{messageInfo{method, id}, args}
	service.socketIO.Write() <- packMsg(&msg)
	log.Infof("[%v] call(): end", service.name)
	return out
}

//Calls a remote method by name and pass args
func (service *Service) Call(name string, args ...interface{}) chan ServiceResult {
	log.Infof("[%v] Call(): start", service.name)
	select {
	case <-service.IsClosed():
		log.Infof("[%v] Call(): start reconnect", service.name)
		if err := service.Reconnect(false); err != nil {
			errorOut := make(chan ServiceResult, 1)
			errorOut <- &serviceRes{nil, &ServiceError{-32, "Disconnected"}}
			return errorOut
		}
	default:
	}
	log.Infof("[%v] Call(): end", service.name)
	return service.call(name, args...)
}

//Disposes resources of a service. You must call this method if the service isn't used anymore.
func (service *Service) Close() {
	log.Infof("[%v] Close(): start", service.name)
	close(service.stop) // Broadcast all related goroutines about disposing
	service.socketIO.Close()
	log.Infof("[%v] Close(): end", service.name)
}

func (service *Service) getServiceChanPair() (In chan ServiceResult, Out chan ServiceResult) {
	In = make(chan ServiceResult)
	Out = make(chan ServiceResult)
	go func() {
		log.Infof("[%v] getServiceChanPair(): start", service.name)
		service.wg.Add(1)
		defer service.wg.Done()
		finished := false
		var pending []ServiceResult
		for {
			log.Infof("[%v] getServiceChanPair: next iteration", service.name)
			var out chan ServiceResult
			var first ServiceResult

			if len(pending) > 0 {
				log.Infof("[%v] getServiceChanPair: changing the value of the first element", service.name)
				first = pending[0]
				out = Out
			} else if finished {
				log.Infof("[%v] getServiceChanPair: finished", service.name)
				close(Out)
				log.Error("[asyncBuff] loop(): output channel is closed")
				break
			}

			select {
			case incoming, ok := <-In:
				if ok {
					log.Infof("[%v] getServiceChanPair: got ServiceResult", service.name)
					pending = append(pending, incoming)
				} else {
					log.Errorf("[%v] getServiceChanPair: input channel is closed", service.name)
					finished = true
					In = nil
				}

			case out <- first:
				log.Infof("[%v] getServiceChanPair: first element has been sent to channel", service.name)
				pending = pending[1:]

			case <-service.stop: // Notification from Close()
				log.Infof("[%v] getServiceChanPair(): Notification from Close()", service.name)
				close(Out)
				log.Infof("[%v] getServiceChanPair(): end", service.name)
				return
			}
		}
		log.Infof("[%v] getServiceChanPair(): end", service.name)
	}()
	return
}
