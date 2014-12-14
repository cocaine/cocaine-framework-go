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
	*Service
}

var (
	locatorServiceInfo = ServiceInfo{
		Endpoints: nil,
		Version:   1,
		API: DispatchMap{
			0: DispatchItem{
				Name:       "resolve",
				Downstream: EmptyDescription,
				Upstream: &StreamDescription{
					0: &StreamDescriptionItem{
						Name:              "write",
						StreamDescription: RecursiveDescription,
					},
					1: &StreamDescriptionItem{
						Name:              "error",
						StreamDescription: EmptyDescription,
					},
				},
			},
		},
	}
)

func NewLocator(args ...string) (*Locator, error) {
	DEBUGTEST("creating locator: %v", args)
	endpoint := DefaultLocator

	if len(args) == 1 {
		endpoint = args[0]
	}

	sock, err := newAsyncConnection("tcp", endpoint, time.Second*5)
	if err != nil {
		DEBUGTEST("unable to create async connection: %s", err)
		return nil, err
	}

	service := Service{
		ServiceInfo:     &locatorServiceInfo,
		socketIO:        sock,
		sessions:        newKeeperStruct(),
		stop:            make(chan struct{}),
		args:            args,
		name:            "locator",
		is_reconnecting: false,
	}
	go service.loop()

	l := &Locator{
		Service: &service,
	}
	return l, nil
}

func (l *Locator) Resolve(name string) (<-chan ResolveChannelResult, error) {
	Out := make(chan ResolveChannelResult, 1)
	channel, err := l.Service.Call("resolve", name)
	if err != nil {
		return nil, err
	}
	DEBUGTEST("After Call in Resolve: %v, %v", channel, err)

	go func() {
		var serviceInfo ServiceInfo
		answer, err := channel.Get()
		if err != nil {
			DEBUGTEST("After channel.Get: %v, %v", answer, err)
		}

		answer.Extract(&serviceInfo)
		Out <- ResolveChannelResult{
			ServiceInfo: &serviceInfo,
			Err:         nil,
		}
	}()

	return Out, nil
}

func (l *Locator) Close() {
	l.socketIO.Close()
}
