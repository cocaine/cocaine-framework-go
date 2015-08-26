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

	setError(error)
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
	if s.err == nil {
		return "<nil>"
	}
	return s.err.Error()
}

func (s *serviceRes) setError(err error) {
	s.err = err
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
	mutex sync.RWMutex
	wg    sync.WaitGroup

	// To keep ordering of opening new sessions
	muKeepSessionOrder sync.Mutex

	socketIO
	*ServiceInfo

	sessions *sessions
	stop     chan struct{}

	args []string
	name string
}

//Creates new service instance with specifed name.
//Optional parameter is a network endpoint of the locator (default ":10053"). Look at Locator.
func serviceResolve(ctx context.Context, name string, endpoints []string) (*ServiceInfo, error) {
	l, err := NewLocator(endpoints)
	if err != nil {
		return nil, err
	}
	defer l.Close()

	return l.Resolve(ctx, name)
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
		return nil, fmt.Errorf("Unable to resolve service %s: %v", name, err)
	}

	sock, err := serviceCreateIO(info.Endpoints)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to service %s: %s", name, err)
	}

	s = &Service{
		socketIO:    sock,
		ServiceInfo: info,
		sessions:    newSessions(),
		stop:        make(chan struct{}),
		args:        endpoints,
		name:        name,
	}
	go s.loop()
	return s, nil
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

func (service *Service) Reconnect(ctx context.Context, force bool) error {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if !force && !service.disconnected() {
		return nil
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
	info, err := serviceResolve(ctx, service.name, service.args)
	if err != nil {
		return err
	}
	sock, err := serviceCreateIO(info.Endpoints)
	if err != nil {
		return err
	}

	// Dispose old IO interface
	service.close()

	// Reattach channels and network IO
	service.stop = make(chan struct{})
	service.socketIO = sock
	// Start service loop
	go service.loop()
	return nil
}

func (service *Service) call(ctx context.Context, name string, args ...interface{}) (Channel, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()

	methodNum, err := service.API.MethodByName(name)
	if err != nil {
		return nil, err
	}

	ch := channel{
		rx: rx{
			pushBuffer: make(chan ServiceResult, 1),
			rxTree:     service.ServiceInfo.API[methodNum].Upstream,
			done:       false,
		},
		tx: tx{
			service: service,
			txTree:  service.ServiceInfo.API[methodNum].Downstream,
			id:      0,
			done:    false,
		},
	}

	// We must create new sessions in the monotonic order
	// Protect sending messages, which open new sessions.
	service.muKeepSessionOrder.Lock()
	defer service.muKeepSessionOrder.Unlock()

	ch.tx.id = service.sessions.Attach(&ch)

	msg := &Message{
		CommonMessageInfo{
			ch.tx.id,
			methodNum},
		args,
	}

	service.sendMsg(msg)
	return &ch, nil
}

func (service *Service) disconnected() bool {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	select {
	case <-service.IsClosed():
		return true
	default:
		return false
	}
}

func (service *Service) sendMsg(msg *Message) {
	service.mutex.RLock()
	service.socketIO.Write() <- msg
	service.mutex.RUnlock()
}

//Calls a remote method by name and pass args
func (service *Service) Call(ctx context.Context, name string, args ...interface{}) (Channel, error) {
	if service.disconnected() {
		if err := service.Reconnect(ctx, false); err != nil {
			return nil, err
		}
	}

	return service.call(ctx, name, args...)
}

// Disposes resources of a service. You must call this method if the service isn't used anymore.
func (service *Service) Close() {
	service.mutex.RLock()
	// Broadcast all related
	// goroutines about disposing
	service.close()
	service.mutex.RUnlock()
}

func (service *Service) close() {
	close(service.stop)
	service.socketIO.Close()
}
