package cocaine

import (
	"fmt"
	"net"
)

type Message struct {
	CommonMessageInfo
	Payload []interface{}
}

type CommonMessageInfo struct {
	// Session id
	Session uint64
	// Message type number
	MsgType uint64
}

// Description of service endpoint
type EndpointItem struct {
	//Service ip address
	IP string
	//Service port
	Port uint64
}

func (e *EndpointItem) String() string {
	return net.JoinHostPort(e.IP, fmt.Sprintf("%d", e.Port))
}

type DispatchMap map[uint64]DispatchItem

type DispatchItem struct {
	Name       string
	Downstream StreamDescription
	Upstream   StreamDescription
}

type StreamDescription map[uint64]struct {
	Name string
	*StreamDescription
}

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
