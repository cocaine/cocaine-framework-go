package cocaine12

import (
	"context"
	"time"
)

// Locator is used to Resolve new services. It should be closed
// after last usage
type Locator interface {
	Resolve(ctx context.Context, name string) (*ServiceInfo, error)
	Close()
}

type locator struct {
	*Service
}

// NewLocator creates a new Locator using given endpoints
func NewLocator(endpoints []string) (Locator, error) {
	if len(endpoints) == 0 {
		endpoints = append(endpoints, GetDefaults().Locators()...)
	}

	var (
		sock socketIO
		err  error
	)

	// ToDo: Duplicated code with Service connection
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
		ServiceInfo: newLocatorServiceInfo(),
		socketIO:    sock,
		sessions:    newSessions(),
		stop:        make(chan struct{}),
		args:        endpoints,
		name:        "locator",
	}
	go service.loop()

	return &locator{
		Service: &service,
	}, nil
}

func (l *locator) Resolve(ctx context.Context, name string) (*ServiceInfo, error) {
	channel, err := l.Service.Call(ctx, "resolve", name)
	if err != nil {
		return nil, err
	}

	answer, err := channel.Get(ctx)
	if err != nil {
		return nil, err
	}

	var serviceInfo ServiceInfo
	if err := answer.Extract(&serviceInfo); err != nil {
		return nil, err
	}

	return &serviceInfo, nil
}

func (l *locator) Close() {
	l.socketIO.Close()
}
