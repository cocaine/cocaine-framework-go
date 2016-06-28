package cocaine

import (
	"errors"
	"fmt"
	"time"

	"github.com/ugorji/go/codec"
)

type Endpoint struct {
	Host string
	Port int
}

func (endpoint *Endpoint) AsString() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

type ResolveResult struct {
	success  bool
	Endpoint `codec:",omitempty"`
	Version  int
	API      map[int64]string
}

func (r *ResolveResult) getMethodNumber(name string) (number int64, err error) {
	for key, value := range r.API {
		if value == name {
			number = key
			return
		}
	}
	err = errors.New("Missing method")
	return
}

type Locator struct {
	unpacker *streamUnpacker
	socketIO
	logger LocalLogger
}

func NewLocator(logger LocalLogger, args ...interface{}) (*Locator, error) {
	endpoint := flagLocator

	if len(args) == 1 {
		if _endpoint, ok := args[0].(string); ok {
			endpoint = _endpoint
		}
	}

	sock, err := newAsyncRWSocket("tcp", endpoint, time.Second*5, logger)
	if err != nil {
		return nil, err
	}
	return &Locator{newStreamUnpacker(), sock, logger}, nil
}

func (locator *Locator) unpackchunk(chunk rawMessage) ResolveResult {
	var res ResolveResult
	err := codec.NewDecoderBytes(chunk, h).Decode(&res)
	if err != nil {
		locator.logger.Errf("unpack chunk error: %v", err)
	}
	return res
}

func (locator *Locator) Resolve(name string) chan ResolveResult {
	Out := make(chan ResolveResult)
	go func() {
		var resolveresult ResolveResult
		resolveresult.success = false
		msg := ServiceMethod{messageInfo{0, 0}, []interface{}{name}}
		locator.socketIO.Write() <- packMsg(&msg)
		closed := false
		for !closed {
			answer := <-locator.socketIO.Read()
			msgs := locator.unpacker.Feed(answer, locator.logger)
			for _, item := range msgs {
				switch id := item.getTypeID(); id {
				case CHUNK:
					resolveresult = locator.unpackchunk(item.getPayload()[0].([]byte))
					resolveresult.success = true
				case CHOKE:
					closed = true
				}
			}
		}
		Out <- resolveresult
	}()
	return Out
}

func (locator *Locator) Close() {
	locator.socketIO.Close()
}
