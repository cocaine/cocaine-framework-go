package cocaine

// import (
// 	"bytes"
// 	"github.com/ugorji/go/codec"
// 	"net/http"
// 	"reflect"
// 	"fmt"
// 	"compress/gzip"
// 	"io"
// 	"errors"
// )

// type Headers [][2]string

// type HTTPReq struct {
// 	*http.Request
// }

// // TBD: Extract more info
// func UnpackProxyRequest(raw []byte) (*http.Request, error) {
// 	var (
// 		mh codec.MsgpackHandle
// 		h  = &mh
// 	)
// 	var v []interface{}
// 	mh.SliceType = reflect.TypeOf(Headers(nil))
// 	codec.NewDecoderBytes(raw, h).Decode(&v)
// 	r, err := http.NewRequest(string(v[0].([]uint8)), string(v[1].([]uint8)), bytes.NewBuffer(v[4].([]byte)))
// 	if err != nil {
// 		return nil, err
// 	}
// 	r.Header = CocaineHeaderToHttpHeader(v[3].(Headers))

// 	r.Host = r.Header.Get("Host")

// 	if xRealIp := r.Header.Get("X-Real-IP"); xRealIp != "" {
// 		r.RemoteAddr = xRealIp
// 	}

// 	err = decompressBody(r)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return r, nil
// }

// func WriteHead(code int, headers Headers) interface{} {
// 	return []interface{}{code, headers}
// }

// func HttpHeaderToCocaineHeader(header http.Header) Headers {
// 	hdr := Headers{}
// 	for headerName, headerValues := range header {
// 		for _, headerValue := range headerValues {
// 			hdr = append(hdr, [2]string{headerName, headerValue})
// 		}
// 	}
// 	return hdr
// }

// func CocaineHeaderToHttpHeader(hdr Headers) http.Header {
// 	header := http.Header{}
// 	for _, hdrValues := range hdr {
// 		header.Add(hdrValues[0], hdrValues[1])
// 	}
// 	return header
// }

// // WrapHandler provides opportunity for using Go web frameworks, which supports http.Handler interface
// //
// //  Trivial example which is used martini web framework
// //
// //	import (
// //		"github.com/cocaine/cocaine-framework-go/cocaine"
// //		"github.com/codegangsta/martini"
// //	)
// //
// //
// //	func main() {
// //		m := martini.Classic()
// //		m.Get("", func() string {
// //				return "This is an example server"
// //			})
// //
// //		m.Get("/hw", func() string {
// //				return "Hello world!"
// //			})
// //
// //		binds := map[string]cocaine.EventHandler{
// //			"example": cocaine.WrapHandler(m, nil),
// //		}
// //		if worker, err := cocaine.NewWorker(); err == nil{
// //			worker.Loop(binds)
// //		}else{
// //			panic(err)
// //		}
// //	}
// //
// func WrapHandler(handler http.Handler, logger *Logger) EventHandler {
// 	var err error
// 	if logger == nil {
// 		logger, err = NewLogger()
// 		if err != nil {
// 			panic(fmt.Sprintf("Could not initialize logger due to error: %v", err))
// 		}
// 	}
// 	var wrapper = func (request *Request, response *Response) {
// 		if httpRequest, err := UnpackProxyRequest(<-request.Read()); err != nil {
// 			logger.Errf("Could not unpack http request due to error %v", err)
// 			response.Write(WriteHead(400, Headers{}))
// 		} else {
// 			w := &ResponseWriter{
// 				cRes: response,
// 				req: httpRequest,
// 				handlerHeader: make(http.Header),
// 				contentLength: -1,
// 				wroteHeader: false,
// 				logger: logger,

// 			}
// 			handler.ServeHTTP(w, httpRequest)
// 			w.finishRequest()
// 		}
// 		response.Close()
// 	}

// 	return wrapper
// }

// // WrapHandlerFunc provides opportunity for using Go web frameworks, which supports http.HandlerFunc interface
// //
// //  Trivial example is
// //
// //	import (
// //		"net/http"
// //		"github.com/cocaine/cocaine-framework-go/cocaine"
// //	)
// //
// //	func handler(w http.ResponseWriter, req *http.Request) {
// //		w.Header().Set("Content-Type", "text/plain")
// //		w.Write([]byte("This is an example server.\n"))
// //	}
// //
// //	func main() {
// //		binds := map[string]cocaine.EventHandler{
// //			"example": cocaine.WrapHandlerFunc(handler, nil),
// //		}
// //		if worker, err := cocaine.NewWorker(); err == nil{
// //			worker.Loop(binds)
// //		}else{
// //			panic(err)
// //		}
// //	}
// //
// func WrapHandlerFunc(hf http.HandlerFunc, logger *Logger) EventHandler {
// 	return WrapHandler(http.HandlerFunc(hf), logger)
// }

// func WrapHandleFuncs(hfs map[string]http.HandlerFunc, logger *Logger) (handlers map[string]EventHandler) {
// 	handlers = map[string]EventHandler{}
// 	for key, hf := range(hfs){
// 		handlers[key] = WrapHandlerFunc(hf, logger)
// 	}
// 	return
// }

// // By default decompressors such as gzip do not close underlying stream
// // so we need to close it manually. This decorator makes it
// type decompressReaderCloser struct {
// 	decompressReader io.Closer
// 	body io.Closer
// }

// func (this decompressReaderCloser) Close() error {
// 	errDec := this.decompressReader.Close()
// 	errBody := this.body.Close()

// 	if errDec != nil {
// 		return errDec
// 	} else {
// 		return errBody
// 	}
// }

// type readerAndCloser struct {
// 	io.Reader
// 	io.Closer
// }

// // If body is compressed it will be decompressed
// // Currently only gzip supported
// func decompressBody(req *http.Request) (err error) {
// 	if req.ContentLength > 0 && req.Header.Get("Content-Encoding") == "gzip" {
// 		gzReader, zerr := gzip.NewReader(req.Body)
// 		if zerr == nil {
// 			req.Header.Del("Content-Encoding")
// 			req.Header.Del("Content-Length")
// 			req.ContentLength = -1
// 			req.Body = readerAndCloser{gzReader, decompressReaderCloser{gzReader, req.Body}}
// 		} else {
// 			gzReader.Close()
// 			req.Body.Close()
// 			err = errors.New(fmt.Sprintf("Could not decompress gzip body due to error %v", zerr))
// 		}
// 	}

// 	return
// }
