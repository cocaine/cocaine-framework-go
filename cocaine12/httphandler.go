package cocaine12

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"

	"golang.org/x/net/context"

	"github.com/cocaine/cocaine-framework-go/pkg/github.com/ugorji/go/codec"
)

var (
	mh codec.MsgpackHandle
	h  = &mh
)

var (
	mhHTTPReq codec.MsgpackHandle
	hHTTPReq  = &mhHTTPReq
)

// Headers are packed as a tuple of tuples HTTP headers from Cocaine
type Headers [][2]string

// type HTTPReq struct {
// 	*http.Request
// }

// UnpackProxyRequest unpacks a HTTPRequest from a serialized cocaine form
func UnpackProxyRequest(raw []byte) (*http.Request, error) {
	var v struct {
		Method  string
		URI     string
		Version string
		Headers Headers
		Body    []byte
	}

	if err := codec.NewDecoderBytes(raw, hHTTPReq).Decode(&v); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(v.Method, v.URI, bytes.NewBuffer(v.Body))
	if err != nil {
		return nil, err
	}

	req.Header = HeadersCocaineToHTTP(v.Headers)
	req.Host = req.Header.Get("Host")

	if xRealIP := req.Header.Get("X-Real-IP"); xRealIP != "" {
		req.RemoteAddr = xRealIP
	}

	// If body is compressed it will be decompressed
	// Inspired by https://github.com/golang/go/blob/master/src/net/http/transport.go#L883
	hasBody := req != nil && req.Method != "HEAD" && req.ContentLength != 0
	if hasBody && req.Header.Get("Content-Encoding") == "gzip" {
		req.Header.Del("Content-Encoding")
		req.Header.Del("Content-Length")
		req.ContentLength = -1
		req.Body = &gzipReader{body: req.Body}
	}

	return req, nil
}

// WriteHead converts the HTTP status code and the headers to the cocaine format
func WriteHead(code int, headers Headers) []byte {
	var out []byte
	codec.NewEncoderBytes(&out, h).Encode([]interface{}{code, headers})
	return out
}

// HeadersHTTPtoCocaine converts net/http.Header to Cocaine representation
func HeadersHTTPtoCocaine(header http.Header) Headers {
	hdr := make(Headers, 0, len(header))
	for headerName, headerValues := range header {
		for _, headerValue := range headerValues {
			hdr = append(hdr, [2]string{headerName, headerValue})
		}
	}
	return hdr
}

// HeadersCocaineToHTTP converts Cocaine representation of headers to net/http.Header
func HeadersCocaineToHTTP(hdr Headers) http.Header {
	header := make(http.Header, len(hdr))
	for _, hdrValues := range hdr {
		header.Add(hdrValues[0], hdrValues[1])
	}
	return header
}

// WrapHandler provides opportunity for using Go web frameworks, which supports http.Handler interface
//
//  Trivial example which is used martini web framework
//
//	import (
//		"github.com/cocaine/cocaine-framework-go/cocaine"
//		"github.com/codegangsta/martini"
//	)
//
//
//	func main() {
//		m := martini.Classic()
//		m.Get("", func() string {
//				return "This is an example server"
//			})
//
//		m.Get("/hw", func() string {
//				return "Hello world!"
//			})
//
//		binds := map[string]cocaine.EventHandler{
//			"example": cocaine.WrapHandler(m, nil),
//		}
//		if worker, err := cocaine.NewWorker(); err == nil{
//			worker.Loop(binds)
//		}else{
//			panic(err)
//		}
//	}
func WrapHandler(handler http.Handler) EventHandler {
	var wrapper = func(ctx context.Context, request Request, response Response) {
		defer response.Close()

		w, httpRequest, err := convertToHTTPFunc(ctx, request, response)
		if err != nil {
			return
		}

		handler.ServeHTTP(w, httpRequest)
		w.finishRequest()
	}

	return wrapper
}

func WrapHTTPFunc(handler func(ctx context.Context, w http.ResponseWriter, req *http.Request)) EventHandler {
	return func(ctx context.Context, request Request, response Response) {
		defer response.Close()

		w, httpRequest, err := convertToHTTPFunc(ctx, request, response)
		if err != nil {
			return
		}

		handler(ctx, w, httpRequest)
		w.finishRequest()
	}
}

func WrapHTTPFuncs(hfs map[string]func(ctx context.Context, w http.ResponseWriter, req *http.Request)) map[string]EventHandler {
	handlers := make(map[string]EventHandler, len(hfs))
	for key, hf := range hfs {
		handlers[key] = WrapHTTPFunc(hf)
	}

	return handlers
}

func convertToHTTPFunc(ctx context.Context, request Request, response Response) (*ResponseWriter, *http.Request, error) {
	// Read the first chunk
	// It consists of method, uri, httpversion, headers, body.
	// They are packed by msgpack
	msg, err := request.Read(ctx)
	if err != nil {
		if ctx.Err() != nil {
			response.Write(WriteHead(http.StatusRequestTimeout, Headers{}))
			response.Write([]byte("request was not received during a timeout"))
			return nil, nil, ctx.Err()
		}

		response.Write(WriteHead(http.StatusBadRequest, Headers{}))
		response.Write([]byte("cannot process request " + err.Error()))
		return nil, nil, err
	}

	httpRequest, err := UnpackProxyRequest(msg)
	if err != nil {
		response.Write(WriteHead(http.StatusBadRequest, Headers{}))
		response.Write([]byte("malformed request"))
		return nil, nil, err
	}

	w := &ResponseWriter{
		cRes:          response,
		req:           httpRequest,
		handlerHeader: make(http.Header),
		contentLength: -1,
		wroteHeader:   false,
	}

	return w, httpRequest, nil
}

// WrapHandlerFunc provides opportunity for using Go web frameworks, which supports http.HandlerFunc interface
//
//  Trivial example is
//
//	import (
//		"net/http"
//		"github.com/cocaine/cocaine-framework-go/cocaine"
//	)
//
//	func handler(w http.ResponseWriter, req *http.Request) {
//		w.Header().Set("Content-Type", "text/plain")
//		w.Write([]byte("This is an example server.\n"))
//	}
//
//	func main() {
//		binds := map[string]cocaine.EventHandler{
//			"example": cocaine.WrapHandlerFunc(handler, nil),
//		}
//		if worker, err := cocaine.NewWorker(); err == nil{
//			worker.Loop(binds)
//		}else{
//			panic(err)
//		}
//	}
//
func WrapHandlerFunc(hf http.HandlerFunc) EventHandler {
	return WrapHandler(http.HandlerFunc(hf))
}

// WrapHandleFuncs wraps the bunch of http.Handlers to allow them handle requests via Worker
func WrapHandleFuncs(hfs map[string]http.HandlerFunc) map[string]EventHandler {
	handlers := make(map[string]EventHandler, len(hfs))
	for key, hf := range hfs {
		handlers[key] = WrapHandlerFunc(hf)
	}

	return handlers
}

// inspired by https://github.com/golang/go/blob/master/src/net/http/transport.go#L1238
// gzipReader wraps a response body so it can lazily
// call gzip.NewReader on the first call to Read
type gzipReader struct {
	body io.ReadCloser // underlying Response.Body
	zr   io.Reader     // lazily-initialized gzip reader
}

func (gz *gzipReader) Read(p []byte) (n int, err error) {
	if gz.zr == nil {
		gz.zr, err = gzip.NewReader(gz.body)
		if err != nil {
			return 0, err
		}
	}
	return gz.zr.Read(p)
}
func (gz *gzipReader) Close() error {
	return gz.body.Close()
}
