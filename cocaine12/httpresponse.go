package cocaine12

import (
	"net/http"
	"strconv"
)

// ResponseWriter implements http.ResponseWriter interface.
// It implements cocaine integration.
type ResponseWriter struct {
	cRes          ResponseStream
	req           *http.Request
	handlerHeader http.Header
	// number of bytes written in body
	written int64
	// explicitly-declared Content-Length; or -1
	contentLength int64
	// status code passed to WriteHeader
	status      int
	wroteHeader bool
}

// Header returns the header map that will be sent by WriteHeader
func (w *ResponseWriter) Header() http.Header {
	return w.handlerHeader
}

// WriteHeader sends an HTTP response header with status code
func (w *ResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}

	w.wroteHeader = true
	w.status = code

	if cl := w.handlerHeader.Get("Content-Length"); cl != "" {
		if v, err := strconv.ParseInt(cl, 10, 64); err == nil && v >= 0 {
			w.contentLength = v
		} else {
			w.handlerHeader.Del("Content-Length")
		}
	}

	w.cRes.Write(
		WriteHead(code, HeadersHTTPtoCocaine(w.handlerHeader)),
	)
}

func (w *ResponseWriter) finishRequest() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.req.MultipartForm != nil {
		w.req.MultipartForm.RemoveAll()
	}

}

// bodyAllowed returns true if a Write is allowed for this response type.
// It's illegal to call this before the header has been flushed.
func (w *ResponseWriter) bodyAllowed() bool {
	if !w.wroteHeader {
		panic("")
	}

	return w.status != http.StatusNotModified
}

// Write writes the data to the connection as part of an HTTP reply
func (w *ResponseWriter) Write(data []byte) (n int, err error) {
	return w.write(len(data), data, "")
}

// WriteString writes the string to the connection as part of an HTTP reply
func (w *ResponseWriter) WriteString(data string) (n int, err error) {
	return w.write(len(data), nil, data)
}

// either dataB or dataS is non-zero.
func (w *ResponseWriter) write(lenData int, dataB []byte, dataS string) (n int, err error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if lenData == 0 {
		return 0, nil
	}

	if !w.bodyAllowed() {
		return 0, http.ErrBodyNotAllowed
	}

	w.written += int64(lenData) // ignoring errors, for errorKludge
	if w.contentLength != -1 && w.written > w.contentLength {
		return 0, http.ErrContentLength
	}

	if dataB != nil {
		w.cRes.Write(dataB)
		return len(dataB), nil
	}

	w.cRes.Write(dataS)
	return len(dataS), nil
}
