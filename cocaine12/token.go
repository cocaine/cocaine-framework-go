package cocaine12

import (
	"sort"
	"sync"
)

var (
	tmMu      sync.RWMutex
	factories = make(map[string]TokenManagerFactory)
)

// Token carries information about an authorization token.
type Token struct {
	ty   string
	body string
}

func NewToken(ty string, body string) Token {
	return Token{ty, body}
}

// Type returns token's type.
// Depending on the context it can be empty, OAUTH, TVM etc.
func (t Token) Type() string {
	return t.ty
}

// Body returns token's body.
// The real body meaning can be determined via token's type.
func (t Token) Body() string {
	return t.body
}

type TokenManagerFactory interface {
	Create(appName string, token Token) (TokenManager, error)
}

// TokenManager automates tokens housekeeping, like timely updation to
// be able to always provide the most fresh tokens.
type TokenManager interface {
	Token() Token
	Stop()
}

// NullTokenManager represents no-op token manager.
// It always returns an empty token.
type NullTokenManager struct{}

func (t *NullTokenManager) Token() Token {
	return Token{}
}

func (t *NullTokenManager) Stop() {}

// TokenManagers returns a sorted list of the names of the registered token
// manager factories.
func TokenManagers() []string {
	tmMu.RLock()
	defer tmMu.RUnlock()

	var list []string
	for name := range factories {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

// Register makes a token manager factory available by the provided name.
// If Register is called twice with the same name or if factory
// is nil, it panics.
func Register(name string, factory TokenManagerFactory) {
	tmMu.Lock()
	defer tmMu.Unlock()

	if factory == nil {
		panic("cocaine: TokenManagerFactory is nil")
	}

	if _, dup := factories[name]; dup {
		panic("cocaine: Register called twice for token manager factory " + name)
	}

	factories[name] = factory
}

func NewTokenManager(appName string, token Token) (TokenManager, error) {
	tmMu.Lock()
	defer tmMu.Unlock()

	if factory, ok := factories[token.ty]; ok {
		return factory.Create(appName, token)
	}

	return new(NullTokenManager), nil
}
