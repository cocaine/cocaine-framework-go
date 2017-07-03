package cocaine12

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/tinylib/msgp/msgp"
)

const (
	ErrDisconnected = -100
)

var (
	// ErrZeroEndpoints returns from serviceCreateIO if passed `endpoints` is an empty array
	ErrZeroEndpoints = errors.New("Endpoints must contain at least one item")
)

// ConnectionError contains an error and an endpoint
type ConnectionError struct {
	EndpointItem
	Err error
}

// MultiConnectionError returns from a connector which iterates over provided endpoints
type MultiConnectionError []ConnectionError

func (m MultiConnectionError) Error() string {
	var b bytes.Buffer
	for _, v := range m {
		b.WriteString(v.EndpointItem.String())
		b.WriteByte(':')
		b.WriteByte(' ')
		b.WriteString(v.Err.Error())
		b.WriteByte(' ')
	}

	return b.String()
}

type ServiceInfo struct {
	Endpoints []EndpointItem
	Version   uint64
	API       dispatchMap
}

type ServiceResult interface {
	Extract(interface{}) error
	ExtractTuple(...interface{}) error
	Result() (uint64, []interface{}, error)
	Err() error

	setError(error)
}

type serviceRes struct {
	payload msgp.Raw
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

func (s *serviceRes) Result() (uint64, []interface{}, error) {
	sz, b, err := msgp.ReadArrayHeaderBytes(s.payload)
	if err != nil {
		return s.method, nil, err
	}

	args := make([]interface{}, 0, sz)
	var value interface{}
	for ; sz > 0; sz-- {
		value, b, err = msgp.ReadIntfBytes(b)
		if err != nil {
			return s.method, nil, err
		}
		args = append(args, value)
	}

	return s.method, args, s.err
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

	epoch uint
	id    string
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

func serviceCreateIO(endpoints []EndpointItem) (socketIO, error) {
	if len(endpoints) == 0 {
		return nil, ErrZeroEndpoints
	}

	var mErr = make(MultiConnectionError, 0)
	for _, endpoint := range endpoints {
		sock, err := newAsyncConnection("tcp", endpoint.String(), time.Second*1)
		if err != nil {
			mErr = append(mErr, ConnectionError{endpoint, err})
			continue
		}

		return sock, nil
	}

	return nil, mErr
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
		epoch:       0,
		id:          fmt.Sprintf("%x", rand.Int63()),
	}
	go s.loop()
	return s, nil
}

func (service *Service) loop() {
	epoch := service.epoch

	for data := range service.socketIO.Read() {
		if rx, ok := service.sessions.Get(data.session); ok {
			rx.push(&serviceRes{
				payload: data.payload,
				method:  data.msgType,
			})
		}
	}

	service.mutex.Lock()
	defer service.mutex.Unlock()
	if epoch == service.epoch {
		service.pushDisconnectedError()
	}
}

func (service *Service) Reconnect(ctx context.Context, force bool) error {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	ctx, closeReconnectionSpan := NewSpan(ctx, "%s %s reconnection", service.name, service.id)
	defer closeReconnectionSpan()

	if !force && !service.disconnected() {
		return nil
	}

	service.pushDisconnectedError()

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
	service.epoch++
	service.socketIO = sock
	// Start service loop
	go service.loop()
	return nil
}

func (service *Service) pushDisconnectedError() {
	for _, key := range service.sessions.Keys() {
		service.sessions.RLock()
		if ch, ok := service.sessions.Get(key); ok {
			ch.push(&serviceRes{
				payload: nil,
				method:  1,
				err:     &ServiceError{ErrDisconnected, "Disconnected"}})
		}
		service.sessions.RUnlock()
		service.sessions.Detach(key)
	}
}

func (service *Service) call(ctx context.Context, name string, args ...interface{}) (Channel, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()

	ctx, traceCall := NewSpan(ctx, "%s %s: calling %s", service.name, service.id, name)

	methodNum, err := service.API.MethodByName(name)
	if err != nil {
		traceCall()
		return nil, err
	}

	var (
		headers           = make(CocaineHeaders)
		traceSentCall     = closeDummySpan
		traceReceivedCall = closeDummySpan
	)

	if traceInfo := getTraceInfo(ctx); traceInfo != nil {

		// eval it once here, to reuse later in traceReceived/Sent
		RPCName := fmt.Sprintf("%s %s: calling %s", service.name, service.id, name)
		traceHex := fmt.Sprintf("%x", traceInfo.trace)
		spanHex := fmt.Sprintf("%x", traceInfo.span)
		parentHex := fmt.Sprintf("%x", traceInfo.parent)

		traceInfoToHeaders(headers, traceInfo)

		traceSentCall = func() {
			traceInfo.getLog().WithFields(Fields{
				"trace_id":       traceHex,
				"span_id":        spanHex,
				"parent_id":      parentHex,
				"real_timestamp": time.Now().UnixNano() / 1000,
				"RPC":            RPCName,
			}).Infof("trace sent")
		}

		traceReceivedCall = func() {
			traceInfo.getLog().WithFields(Fields{
				"trace_id":       traceHex,
				"span_id":        spanHex,
				"parent_id":      parentHex,
				"real_timestamp": time.Now().UnixNano() / 1000,
				"RPC":            RPCName,
			}).Infof("trace received")
		}
	}

	ch := channel{
		traceReceived: traceReceivedCall,
		traceSent:     traceSentCall,
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
			headers: headers,
		},
	}

	// We must create new sessions in the monotonic order
	// Protect sending messages, which open new sessions.
	service.muKeepSessionOrder.Lock()
	defer service.muKeepSessionOrder.Unlock()

	ch.tx.id = service.sessions.Attach(&ch)

	msg, err := newMessage(ch.tx.id, methodNum, args, headers)
	if err != nil {
		return nil, err
	}
	service.sendMsg(msg)
	return &ch, nil
}

func (service *Service) disconnected() bool {
	select {
	case <-service.IsClosed():
		return true
	default:
		return false
	}
}

func (service *Service) sendMsg(msg *message) {
	service.mutex.RLock()
	service.socketIO.Send(msg)
	service.mutex.RUnlock()
}

// Call a remote method by name and pass args
func (service *Service) Call(ctx context.Context, name string, args ...interface{}) (Channel, error) {
	service.mutex.RLock()
	disconnected := service.disconnected()
	service.mutex.RUnlock()

	if disconnected {
		if err := service.Reconnect(ctx, false); err != nil {
			return nil, err
		}
	}

	return service.call(ctx, name, args...)
}

// Close disposes resources of a service. You must call this method if the service isn't used anymore.
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
