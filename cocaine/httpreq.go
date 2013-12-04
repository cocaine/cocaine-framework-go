package cocaine

import (
	"bytes"
	"github.com/ugorji/go/codec"
	"net/http"
	"reflect"
)

type Headers [][2]string

type HTTPReq struct {
	*http.Request
}

// TBD: Extract more info
func UnpackProxyRequest(raw []byte) (*http.Request, error) {
	var (
		mh codec.MsgpackHandle
		h  = &mh
	)
	var v []interface{}
	mh.SliceType = reflect.TypeOf(Headers(nil))
	codec.NewDecoderBytes(raw, h).Decode(&v)
	r, err := http.NewRequest(string(v[0].([]uint8)), string(v[1].([]uint8)), bytes.NewBuffer(v[4].([]byte)))
	if err != nil {
		return nil, err
	}
	r.Header = CocaineHeaderToHttpHeader(v[3].(Headers))

	r.Host = r.Header.Get("Host")

	if xRealIp := r.Header.Get("X-Real-IP"); xRealIp != "" {
		r.RemoteAddr = xRealIp
	}
	return r, nil
}

func WriteHead(code int, headers Headers) interface{} {
	return []interface{}{code, headers}
}

func HttpHeaderToCocaineHeader(header http.Header) Headers {
	hdr := Headers{}
	for headerName, headerValues := range header {
		for _, headerValue := range headerValues {
			hdr = append(hdr, [2]string{headerName, headerValue})
		}
	}
	return hdr
}

func CocaineHeaderToHttpHeader(hdr Headers) http.Header {
	header := http.Header{}
	for _, hdrValues := range hdr {
		header.Add(hdrValues[0], hdrValues[1])
	}
	return header
}
