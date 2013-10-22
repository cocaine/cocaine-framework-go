package cocaine

import (
	"fmt"
	"github.com/ugorji/go/codec"
	"log"
	"reflect"
	"time"
)

type Endpoint struct {
	Host string
	Port uint64
}

func (endpoint *Endpoint) AsString() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

type ResolveResult struct {
	success bool
	Endpoint
	Version int64
	API     map[int]string
}

//NewLocator

type Locator struct {
	host     string
	port     uint64
	unpacker *StreamUnpacker
	AsyncIO
}

func NewLocator(host string, port uint64) (*Locator, error) {
	sock, err := NewASocket("tcp", fmt.Sprintf("%s:%d", host, port), time.Second*5)
	if err != nil {
		return nil, err
	}
	return &Locator{host, port, NewStreamUnpacker(), sock}, nil
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
		return ResolveResult{true, endpoint, version, api}
	} else {
		panic("Bad format")
	}
}

func (locator *Locator) Resolve(name string) chan ResolveResult {
	Out := make(chan ResolveResult)
	go func() {
		var resolveresult ResolveResult
		resolveresult.success = false
		msg := ServiceMethod{MessageInfo{0, 0}, []interface{}{name}}
		raw := Pack(&msg)
		locator.AsyncIO.Write() <- raw
		closed := false
		for !closed {
			answer := <-locator.AsyncIO.Read()
			msgs := locator.unpacker.Feed(answer) //DecodeRaw(answer, &left)
			for _, item := range msgs {
				switch id := item.GetTypeID(); id {
				case CHUNK:
					resolveresult = locator.unpackchunk(item.GetPayload()[0].([]byte))
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
	locator.AsyncIO.Close()
}
