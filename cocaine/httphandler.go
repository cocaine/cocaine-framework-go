package cocaine

import (
	"bytes"
	"compress/gzip"
	// "errors"
	// "fmt"
	"io"
	"net/http"

	"github.com/cocaine/cocaine-framework-go/vendor/src/github.com/ugorji/go/codec"
)

var (
	mhHTTPReq codec.MsgpackHandle
	hHTTPReq  = &mhHTTPReq
)

type Headers [][2]string

type HTTPReq struct {
	*http.Request
}

// TBD: Extract more info
func UnpackProxyRequest(raw []byte) (*http.Request, error) {
	var v struct {
		Method  string
		Uri     string
		Version string
		Headers Headers
		Body    []byte
	}

	codec.NewDecoderBytes(raw, hHTTPReq).Decode(&v)
	req, err := http.NewRequest(v.Method, v.Uri, bytes.NewBuffer(v.Body))
	if err != nil {
		return nil, err
	}

	req.Header = CocaineHeaderToHttpHeader(v.Headers)
	req.Host = req.Header.Get("Host")

	if xRealIp := req.Header.Get("X-Real-IP"); xRealIp != "" {
		req.RemoteAddr = xRealIp
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

func WriteHead(code int, headers Headers) interface{} {
	return []interface{}{code, headers}
}

func HttpHeaderToCocaineHeader(header http.Header) Headers {
	hdr := make(Headers, len(header))
	for headerName, headerValues := range header {
		for _, headerValue := range headerValues {
			hdr = append(hdr, [2]string{headerName, headerValue})
		}
	}
	return hdr
}

func CocaineHeaderToHttpHeader(hdr Headers) http.Header {
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
//
// func WrapHandler(handler http.Handler, logger *Logger) EventHandler {
// 	var err error
// 	if logger == nil {
// 		logger, err = NewLogger()
// 		if err != nil {
// 			panic(fmt.Sprintf("Could not initialize logger due to error: %v", err))
// 		}
// 	}
// 	var wrapper = func(request *Request, response *Response) {
// 		if httpRequest, err := UnpackProxyRequest(<-request.Read()); err != nil {
// 			logger.Errf("Could not unpack http request due to error %v", err)
// 			response.Write(WriteHead(400, Headers{}))
// 		} else {
// 			w := &ResponseWriter{
// 				cRes:          response,
// 				req:           httpRequest,
// 				handlerHeader: make(http.Header),
// 				contentLength: -1,
// 				wroteHeader:   false,
// 				// logger:        logger,
// 			}
// 			handler.ServeHTTP(w, httpRequest)
// 			w.finishRequest()
// 		}
// 		response.Close()
// 	}

// 	return wrapper
// }

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
// func WrapHandlerFunc(hf http.HandlerFunc, logger *Logger) EventHandler {
// 	return WrapHandler(http.HandlerFunc(hf), logger)
// }

// func WrapHandleFuncs(hfs map[string]http.HandlerFunc, logger *Logger) (handlers map[string]EventHandler) {
// 	handlers = map[string]EventHandler{}
// 	for key, hf := range hfs {
// 		handlers[key] = WrapHandlerFunc(hf, logger)
// 	}
// 	return
// }

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
