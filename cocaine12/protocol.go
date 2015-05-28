package cocaine12

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

// EndpointItem is one of possible endpoints of a service
type EndpointItem struct {
	// Service ip address
	IP string
	// Service port
	Port uint64
}

func (e *EndpointItem) String() string {
	return net.JoinHostPort(e.IP, fmt.Sprintf("%d", e.Port))
}

type DispatchMap map[uint64]DispatchItem

type DispatchItem struct {
	Name       string
	Downstream *StreamDescription
	Upstream   *StreamDescription
}

type StreamDescription map[uint64]*StreamDescriptionItem

func (s *StreamDescription) MethodByName(name string) (uint64, error) {
	for i, v := range *s {
		if v.Name == name {
			return i, nil
		}
	}

	return 0, fmt.Errorf("no `%s` method", name)
}

type StreamDescriptionItem struct {
	Name string
	*StreamDescription
}

var (
	EmptyDescription     = &StreamDescription{}
	RecursiveDescription *StreamDescription
)

func (d *DispatchMap) Methods() []string {
	var methods = make([]string, 0, len(*d))
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
	return 0, fmt.Errorf("no `%s` method", name)
}
