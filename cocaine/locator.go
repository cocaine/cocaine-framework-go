package cocaine

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/cocaine/cocaine-framework-go/vendor/src/github.com/ugorji/go/codec"
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
}

func NewLocator(args ...interface{}) (*Locator, error) {
	if !flag.Parsed() {
		flag.Parse()
	}
	endpoint := flagLocator

	if len(args) == 1 {
		if _endpoint, ok := args[0].(string); ok {
			endpoint = _endpoint
		}
	}

	sock, err := newAsyncRWSocket("tcp", endpoint, time.Second*5)
	if err != nil {
		return nil, err
	}
	return &Locator{newStreamUnpacker(), sock}, nil
}

func (locator *Locator) unpackchunk(chunk rawMessage) ResolveResult {
	var res ResolveResult
	err := codec.NewDecoderBytes(chunk, h).Decode(&res)
	if err != nil {
		log.Println("unpack chunk error", err)
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
			msgs := locator.unpacker.Feed(answer)
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
