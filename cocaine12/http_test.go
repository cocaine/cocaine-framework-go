package cocaine

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"testing"

	"github.com/cocaine/cocaine-framework-go/vendor/src/github.com/ugorji/go/codec"
)

var (
	method  = "GET"
	uri     = "/path?a=1"
	version = "1.1"
	headers = []interface{}{
		[2]string{"Content-Type", "text/html"},
		[2]string{"X-Cocaine-Service", "Test"},
		[2]string{"X-Cocaine-Service", "Test2"},
	}
	body = gzipedBody()
	req  = []interface{}{method, uri, version, headers, body}
)

func gzipedBody() []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write([]byte("hello, world\n"))
	w.Close()
	return b.Bytes()
}

func TestHTTPDecoder(t *testing.T) {

	var out []byte
	if err := codec.NewEncoderBytes(&out, h).Encode(req); err != nil {
		t.Fatalf("unable to pack test data: %v", err)
	}

	r, err := UnpackProxyRequest(out)
	if err != nil {
		t.Fatalf("unable to unpack request %v", err)
	}
	defer r.Body.Close()

	if b, _ := ioutil.ReadAll(r.Body); !bytes.Equal(b, body) {
		t.Fatalf("bad bytes: %s %s", b, body)
	}

	if r.Method != method {
		t.Fatalf("bad method: %s %s", r.Method, method)
	}

	if r.Header.Get("X-Cocaine-Service") != "Test" {
		t.Fatalf("bad header", r.Header.Get("X-Cocaine-Service"))
	}
}

func BenchmarkHTTPDecoder(b *testing.B) {
	var out []byte
	codec.NewEncoderBytes(&out, h).Encode(req)

	for n := 0; n < b.N; n++ {
		UnpackProxyRequest(out)
	}
}
