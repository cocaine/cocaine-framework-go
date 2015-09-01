package cocaine12

import (
	"fmt"
	"net"
)

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
