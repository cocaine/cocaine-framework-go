package cocaine12

import (
	"time"
)

type ResolveChannelResult struct {
	*ServiceInfo
	Err error
}

type Locator struct {
	*Service
}

func NewLocator(args ...string) (*Locator, error) {
	endpoint := DefaultLocator

	if len(args) == 1 {
		endpoint = args[0]
	}

	sock, err := newAsyncConnection("tcp", endpoint, time.Second*5)
	if err != nil {
		return nil, err
	}

	service := Service{
		ServiceInfo:    newLocatorServiceInfo(),
		socketIO:       sock,
		sessions:       newSessions(),
		stop:           make(chan struct{}),
		args:           args,
		name:           "locator",
		isReconnecting: false,
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
