package tvm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cocaine/cocaine-framework-go/cocaine12"
)

const (
	tvmTokenType        = "TVM"
	tokenRefreshTimeout = time.Second * 15
)

func init() {
	cocaine12.Register(tvmTokenType, new(TicketVendingMachineTokenManagerFactory))
}

type TicketVendingMachineTokenManagerFactory struct{}

func (f *TicketVendingMachineTokenManagerFactory) Create(appName string, token cocaine12.Token) (cocaine12.TokenManager, error) {
	ctx := context.Background()
	tvm, err := cocaine12.NewService(ctx, "tvm", nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	t := &TicketVendingMachineTokenManager{
		appName: appName,
		ticker:  time.NewTicker(tokenRefreshTimeout),
		ctx:     ctx,
		cancel:  cancel,
		mu:      sync.Mutex{},
		ticket:  token,
		tvm:     tvm,
	}

	go t.run()

	return t, nil
}

// TicketVendingMachineTokenManager manages TVM tickets.
type TicketVendingMachineTokenManager struct {
	appName string
	ticker  *time.Ticker
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	ticket  cocaine12.Token
	tvm     *cocaine12.Service
}

func (t *TicketVendingMachineTokenManager) Token() cocaine12.Token {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ticket
}

func (t *TicketVendingMachineTokenManager) Stop() {
	t.cancel()
}

func (t *TicketVendingMachineTokenManager) run() {
	for {
		select {
		case <-t.ticker.C:
			t.mu.Lock()
			body := t.ticket.Body()
			t.mu.Unlock()

			ch, err := t.tvm.Call(t.ctx, "refresh_ticket", t.appName, body)
			if err != nil {
				fmt.Printf("failed to update ticket %v\n", err)
				continue
			}

			answer, err := ch.Get(t.ctx)
			if err != nil {
				fmt.Printf("failed to get ticket %v\n", err)
				continue
			}

			var ticketResult string
			if err := answer.ExtractTuple(&ticketResult); err != nil {
				fmt.Printf("failed to extract ticket %v\n", err)
				continue
			}

			t.mu.Lock()
			t.ticket = cocaine12.NewToken(tvmTokenType, ticketResult)
			t.mu.Unlock()
		case <-t.ctx.Done():
			t.ticker.Stop()
			return
		}
	}
}
