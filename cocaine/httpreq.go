package cocaine

import (
	"bytes"
	"github.com/ugorji/go/codec"
	"fmt"
	"net/http"
	"reflect"
)

type Headers [][2]string

type HTTPReq struct {
	*http.Request
}

// func (req *HTTPReq) args() map[string][]string {

// }

// TBD: Extract more info
func UnpackProxyRequest(raw []byte) *http.Request {
	var (
		mh codec.MsgpackHandle
		h  = &mh
	)
	var v []interface{}
	mh.SliceType = reflect.TypeOf(Headers(nil))
	codec.NewDecoderBytes(raw, h).Decode(&v)
	r, err := http.NewRequest(string(v[0].([]uint8)), string(v[1].([]uint8)), bytes.NewBuffer(v[4].([]byte)))
	if err != nil {
		fmt.Println("Error", err)
	}
	for _, item := range v[3].(Headers) {
		r.Header.Set(item[0], item[1])
		if item[0] == "X-Real-IP" {
			r.RemoteAddr = item[1]
		}
	}
	return r
}

func WriteHead(code int, headers [][2]string) interface{} {
	return []interface{}{code, headers}
}
