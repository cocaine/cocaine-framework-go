package cocaine

import (
	"codec"
	"fmt"
	"log"
	"reflect"
)

type Endpoint struct {
	Host string
	Port uint64
}

func (endpoint *Endpoint) AsString() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

type ResolveResult struct {
	Endpoint
	Version int64
	API     map[int]string
}

type Locator struct {
	host     string
	port     uint64
	pipe     *Pipe
	unpacker *StreamUnpacker
	wr_in    chan RawMessage
	r_out    chan RawMessage
}

func NewLocator(host string, port uint64) *Locator {
	wr_in, wr_out := Transmite()
	r_in, r_out := Transmite()
	pipe := NewPipe("tcp", fmt.Sprintf("%s:%d", host, port), &wr_out, &r_in)
	return &Locator{host, port, pipe, NewStreamUnpacker(), wr_in, r_out}
}

func (locator *Locator) unpackchunk(chunk RawMessage) ResolveResult {
	defer func() {
		if err := recover(); err != nil {
			log.Println("defer", err)
		}
	}()
	var v []interface{}
	var MH codec.MsgpackHandle
	MH.MapType = reflect.TypeOf(map[int]string(nil))
	err := codec.NewDecoderBytes(chunk, &MH).Decode(&v)
	if err != nil {
		log.Println("unpack chunk error", err)
	}
	if len(v) == 3 {
		v_endpoint := v[0].([]interface{})
		endpoint := Endpoint{string(v_endpoint[0].([]uint8)), v_endpoint[1].(uint64)}
		version := v[1].(int64)
		api := v[2].(map[int]string)
		return ResolveResult{endpoint, version, api}
	} else {
		panic("Bad format")
	}
}

func (locator *Locator) Resolve(name string) chan ResolveResult {
	Out := make(chan ResolveResult)
	go func() {
		var resolveresult ResolveResult
		msg := ServiceMethod{MessageInfo{0, 0}, []interface{}{name}}
		raw := Pack(&msg)
		locator.wr_in <- raw
		closed := false
		for !closed {
			answer := <-locator.r_out
			msgs := locator.unpacker.Feed(answer) //DecodeRaw(answer, &left)
			for _, item := range msgs {
				switch id := item.GetTypeID(); id {
				case CHUNK:
					resolveresult = locator.unpackchunk(item.GetPayload()[0].([]byte))
				case CHOKE:
					closed = true
				}
			}
		}
		Out <- resolveresult
	}()
	return Out
}
