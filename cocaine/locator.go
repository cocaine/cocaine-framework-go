package cocaine

import (
	"flag"
	"fmt"
	"net"
	"time"
)

// General message information
type Message struct {
	// Session id
	Session uint64
	// Message type number
	MsgType uint64
}

type GeneralMessage struct {
	Message
	Payload []interface{}
}

type ResolveResult struct {
	Endpoints []EndpointItem
	Version   uint64
	API       DispatchMap
}

type ResolveChannelResult struct {
	*ResolveResult
	Err error
}

// Description of service endpoint
// It consists of ("IP", Port)
type EndpointItem struct {
	//Service ip address
	IP string
	//Service port
	Port uint64
}

func (e *EndpointItem) String() string {
	return net.JoinHostPort(e.IP, fmt.Sprintf("%d", e.Port))
}

type StreamStruct map[uint64]struct {
	Name string
	*StreamStruct
}

type DispatchItem struct {
	Name       string
	Downstream StreamStruct
	Upstream   StreamStruct
}

type DispatchMap map[uint64]DispatchItem

func (d *DispatchMap) Methods() []string {
	var methods []string = make([]string, 0)
	for _, v := range *d {
		methods = append(methods, v.Name)
	}
	return methods
}

func (d *DispatchMap) MethodByName(name string) (uint64, error) {
	for i, v := range *d {
		if v.Name == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no method `%s`", name)
}

type Locator struct {
	socketIO
}

func NewLocator(args ...interface{}) (*Locator, error) {
	if !flag.Parsed() {
		flag.Parse()
	}
	endpoint := flagLocator

	if len(args) == 1 {
		if _endpoint, ok := args[0].(string); ok {
			endpoint = _endpoint
		}
	}

	sock, err := newAsyncRWSocket("tcp", endpoint, time.Second*5)
	if err != nil {
		return nil, err
	}
	return &Locator{
		socketIO: sock,
	}, nil
}

func (locator *Locator) Resolve(name string) <-chan ResolveChannelResult {
	Out := make(chan ResolveChannelResult, 1)
	go func() {
		var resolveresult ResolveResult
		locator.socketIO.Write() <- GeneralMessage{Message{1, 0}, []interface{}{name}}
		answer := <-locator.socketIO.Read()
		// ToDO: Optimize converter like in `mapstructure`
		convertPayload(answer.Payload, &resolveresult)
		Out <- ResolveChannelResult{
			ResolveResult: &resolveresult,
			Err:           nil,
		}
	}()
	return Out
}

func (locator *Locator) Close() {
	locator.socketIO.Close()
}
