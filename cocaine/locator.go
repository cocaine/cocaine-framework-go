package cocaine

import (
	"time"
)

type ServiceInfo struct {
	Endpoints []EndpointItem
	Version   uint64
	API       DispatchMap
}

type ResolveChannelResult struct {
	*ServiceInfo
	Err error
}

type Locator struct {
	socketIO
}

func NewLocator(args ...interface{}) (*Locator, error) {
	endpoint := DefaultLocator

	if len(args) == 1 {
		if _endpoint, ok := args[0].(string); ok {
			endpoint = _endpoint
		}
	}

	sock, err := newAsyncConnection("tcp", endpoint, time.Second*5)
	if err != nil {
		return nil, err
	}

	l := &Locator{
		socketIO: sock,
	}
	return l, nil
}

func (locator *Locator) Resolve(name string) <-chan ResolveChannelResult {
	Out := make(chan ResolveChannelResult, 1)
	go func() {
		var serviceInfo ServiceInfo
		locator.socketIO.Write() <- &Message{
			CommonMessageInfo{1, 0},
			[]interface{}{name},
		}
		answer := <-locator.socketIO.Read()
		// ToDO: Optimize converter like in `mapstructure`
		convertPayload(answer.Payload, &serviceInfo)
		Out <- ResolveChannelResult{
			ServiceInfo: &serviceInfo,
			Err:         nil,
		}
	}()
	return Out
}

func (locator *Locator) Close() {
	locator.socketIO.Close()
}
