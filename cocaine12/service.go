package cocaine12

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/net/context"
)

type ServiceInfo struct {
	Endpoints []EndpointItem
	Version   uint64
	API       DispatchMap
}

type ServiceResult interface {
	Extract(interface{}) error
	ExtractTuple(...interface{}) error
	Result() (uint64, []interface{}, error)
	Err() error
}

type serviceRes struct {
	payload []interface{}
	method  uint64
	err     error
}

//Unpacks the result of the called method in the passed structure.
//You can transfer the structure of a particular type that will avoid the type checking. Look at examples.
func (s *serviceRes) Extract(target interface{}) (err error) {
	if s.err != nil {
		return s.err
	}
	return convertPayload(s.payload, target)
}

func (s *serviceRes) ExtractTuple(args ...interface{}) error {
	return s.Extract(&args)
}

// ToDo: Extract method for an array semantic
// Extract(target ...interface{})

func (s *serviceRes) Result() (uint64, []interface{}, error) {
	return s.method, s.payload, s.err
}

//Error status
func (s *serviceRes) Err() error {
	return s.err
}

func (s *serviceRes) Error() string {
	return s.err.Error()
}

//
type ServiceError struct {
	Code    int
	Message string
}

func (err *ServiceError) Error() string {
	return err.Message
}

// Allows you to invoke methods of services and send events to other cloud applications.
type Service struct {
	// Tracking a connection state
	mutex sync.Mutex
	wg    sync.WaitGroup

	socketIO
	*ServiceInfo

	sessions *sessions
	stop     chan struct{}

	args           []string
	name           string
	isReconnecting bool
}

//Creates new service instance with specifed name.
//Optional parameter is a network endpoint of the locator (default ":10053"). Look at Locator.
func serviceResolve(ctx context.Context, name string, endpoints []string) (info *ServiceInfo, err error) {
	l, err := NewLocator(endpoints)
	if err != nil {
		return
	}
	defer l.Close()

	ch, err := l.Resolve(ctx, name)
	if err != nil {
		return nil, err
	}

	serviceInfo := <-ch
	info = serviceInfo.ServiceInfo
	err = serviceInfo.Err
	return
}

func serviceCreateIO(endpoints []EndpointItem) (sock socketIO, err error) {
CONN_LOOP:
	for _, endpoint := range endpoints {
		sock, err = newAsyncConnection("tcp", endpoint.String(), time.Second*1)
		if err != nil {
			continue
		}

		break CONN_LOOP
	}

	return
}

func NewService(ctx context.Context, name string, endpoints []string) (s *Service, err error) {
	info, err := serviceResolve(ctx, name, endpoints)
	if err != nil {
		err = fmt.Errorf("Unable to resolve service %s", name)
		return
	}

	sock, err := serviceCreateIO(info.Endpoints)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to service %s: %s", name, err)
	}

	s = &Service{
		socketIO:       sock,
		ServiceInfo:    info,
		sessions:       newSessions(),
		stop:           make(chan struct{}),
		args:           endpoints,
		name:           name,
		isReconnecting: false,
	}
	go s.loop()
	return
}

func (service *Service) loop() {
	for data := range service.socketIO.Read() {
		if rx, ok := service.sessions.Get(data.Session); ok {
			rx.push(&serviceRes{
				payload: data.Payload,
				method:  data.MsgType,
			})
		}
	}
}

func (service *Service) Reconnect(force bool) error {
	if !service.isReconnecting {
		service.mutex.Lock()
		defer service.mutex.Unlock()

		if service.isReconnecting {
			return fmt.Errorf("%s", "Service is reconnecting now")
		}

		service.isReconnecting = true
		defer func() { service.isReconnecting = false }()

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
			if ch, ok := service.sessions.Get(key); ok {
				ch.push(&serviceRes{
					payload: nil,
					method:  1,
					err:     &ServiceError{-100, "Disconnected"}})
			}
			service.sessions.RUnlock()
			service.sessions.Detach(key)
		}

		// Create new socket
		info, err := serviceResolve(context.Background(), service.name, service.args)
		if err != nil {
			return err
		}
		sock, err := serviceCreateIO(info.Endpoints)
		if err != nil {
			return err
		}

		// Dispose old IO interface
		service.Close()

		// Reattach channels and network IO
		service.stop = make(chan struct{})
		service.socketIO = sock
		// Start service loop
		go service.loop()
		return nil
	}
	return fmt.Errorf("Service is reconnecting now")
}

func (service *Service) call(name string, args ...interface{}) (Channel, error) {
	methodNum, err := service.API.MethodByName(name)
	if err != nil {
		return nil, err
	}

	ch := channel{
		rx: rx{
			pushBuffer: make(chan ServiceResult, 1),
			rxTree:     service.ServiceInfo.API[methodNum].Upstream,
		},
		tx: tx{
			service: service,
			txTree:  service.ServiceInfo.API[methodNum].Downstream,
			id:      0,
			done:    false,
		},
	}

	ch.tx.id = service.sessions.Attach(&ch)

	// FIX THIS!!!
	msg := &Message{
		CommonMessageInfo{
			ch.tx.id,
			methodNum},
		args,
	}

	service.sendMsg(msg)
	return &ch, nil
}

func (service *Service) sendMsg(msg *Message) {
	service.socketIO.Write() <- msg
}

//Calls a remote method by name and pass args
func (service *Service) Call(name string, args ...interface{}) (Channel, error) {
	select {
	case <-service.IsClosed():
		if err := service.Reconnect(false); err != nil {
			return nil, err
		}
	default:
	}
	return service.call(name, args...)
}

//Disposes resources of a service. You must call this method if the service isn't used anymore.
func (service *Service) Close() {
	// Broadcast all related goroutines about disposing
	close(service.stop)
	service.socketIO.Close()
}

func (service *Service) getServiceChanPair() (input chan ServiceResult, output chan ServiceResult) {
	input, output = make(chan ServiceResult), make(chan ServiceResult)

	service.wg.Add(1)
	go func() {
		defer service.wg.Done()

		// Notify a receiver
		defer close(output)

		var (
			finished = false
			pending  []ServiceResult
		)

		for {
			var (
				out   chan ServiceResult
				in    = input
				first ServiceResult
			)

			if len(pending) > 0 {
				first = pending[0]
				out = output
			} else if finished {
				break
			}

			select {
			case incoming, ok := <-in:
				if ok {
					pending = append(pending, incoming)
				} else {
					finished = true
					in = nil
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
