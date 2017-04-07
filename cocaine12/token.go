package cocaine12

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	tvmTokenType        = "TVM"
	tokenRefreshTimeout = time.Second * 5
)

type Token struct {
	ty   string
	body string
}

func (t *Token) Type() string {
	return t.ty
}

func (t *Token) Body() string {
	return t.body
}

type TokenManager interface {
	Token() Token
	Stop()
}

type NullTokenManager struct{}

func (t *NullTokenManager) Token() Token {
	return Token{}
}

func (t *NullTokenManager) Stop() {}

type TicketVendingMachineTokenManager struct {
	appName string
	ticker  *time.Ticker
	stopped chan struct{}
	mu      sync.Mutex
	ticket  *Token
	tvm     *Service
}

func (t *TicketVendingMachineTokenManager) Token() Token {
	t.mu.Lock()
	defer t.mu.Unlock()
	return *t.ticket
}

func (t *TicketVendingMachineTokenManager) Stop() {
	close(t.stopped)
}

func (t *TicketVendingMachineTokenManager) run() {
	for {
		select {
		case <-t.ticker.C:
			t.mu.Lock()
			body := t.ticket.Body()
			t.mu.Unlock()

			ctx := context.Background()
			ch, err := t.tvm.Call(ctx, "refresh_ticket", t.appName, body)
			if err != nil {
				fmt.Printf("failed to update ticket %v\n", err)
				continue
			}

			answer, err := ch.Get(ctx)
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
			t.ticket = &Token{tvmTokenType, ticketResult}
			t.mu.Unlock()
		case <-t.stopped:
			t.ticker.Stop()
			return
		}
	}
}

func newTokenManager(appName string, token *Token) (TokenManager, error) {
	if token.ty == tvmTokenType {
		ctx := context.Background()
		tvm, err := NewService(ctx, "tvm", nil)
		if err != nil {
			return nil, err
		}

		t := &TicketVendingMachineTokenManager{
			appName: appName,
			ticker:  time.NewTicker(tokenRefreshTimeout),
			stopped: make(chan struct{}),
			mu:      sync.Mutex{},
			ticket:  token,
			tvm:     tvm,
		}

		go t.run()

		return t, nil
	}

	return new(NullTokenManager), nil
}
