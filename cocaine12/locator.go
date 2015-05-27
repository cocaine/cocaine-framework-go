package cocaine12

import (
	"time"

	"golang.org/x/net/context"
)

type ResolveChannelResult struct {
	*ServiceInfo
	Err error
}

type Locator struct {
	*Service
}

func NewLocator(endpoints []string) (*Locator, error) {
	if len(endpoints) == 0 {
		endpoints = append(endpoints, DefaultLocator)
	}

	var (
		sock socketIO
		err  error
	)

CONN_LOOP:
	for _, endpoint := range endpoints {
		sock, err = newAsyncConnection("tcp", endpoint, time.Second*1)
		if err != nil {
			continue
		}
		break CONN_LOOP
	}

	if err != nil {
		return nil, err
	}

	service := Service{
		ServiceInfo:    newLocatorServiceInfo(),
		socketIO:       sock,
		sessions:       newSessions(),
		stop:           make(chan struct{}),
		args:           endpoints,
		name:           "locator",
		isReconnecting: false,
	}
	go service.loop()

	l := &Locator{
		Service: &service,
	}
	return l, nil
}

func (l *Locator) Resolve(ctx context.Context, name string) (<-chan ResolveChannelResult, error) {
	Out := make(chan ResolveChannelResult, 1)
	channel, err := l.Service.Call("resolve", name)
	if err != nil {
		return nil, err
	}

	go func() {
		answer, err := channel.Get()
		if err != nil {
			Out <- ResolveChannelResult{
				ServiceInfo: nil,
				Err:         err,
			}
			return
		}

		var serviceInfo ServiceInfo
		if err := answer.Extract(&serviceInfo); err != nil {
			Out <- ResolveChannelResult{
				ServiceInfo: nil,
				Err:         err,
			}
		}

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
