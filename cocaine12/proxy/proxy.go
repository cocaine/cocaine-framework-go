package proxy

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
	"github.com/ugorji/go/codec"
	"golang.org/x/net/context"
)

var (
	mhAsocket = codec.MsgpackHandle{
		BasicHandle: codec.BasicHandle{
			EncodeOptions: codec.EncodeOptions{
				StructToArray: true,
			},
		},
	}
	hAsocket = &mhAsocket
)

func packRequest(req *http.Request) ([]byte, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err

	}
	// method uri 1.1 headers body
	headers := make([][2]string, 0, len(req.Header))
	for header, values := range req.Header {
		for _, val := range values {
			headers = append(headers, [2]string{header, val})
		}
	}

	var task []byte
	codec.NewEncoderBytes(&task, hAsocket).Encode([]interface{}{
		req.Method,
		req.URL.RequestURI(),
		fmt.Sprintf("%d.%d", req.ProtoMajor, req.ProtoMinor),
		headers,
		body,
	})

	return task, nil
}

func process(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Powered-By", "Cocaine")
	defer r.Body.Close()
	var (
		service = r.Header.Get("X-Cocaine-Service")
		event   = r.Header.Get("X-Cocaine-Event")
	)

	if len(service) == 0 || len(event) == 0 {
		var (
			j = 1
		)
		npos := strings.IndexByte(r.URL.Path[j:], '/')
		if npos < 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		service = r.URL.Path[j : npos+j]
		j += npos + 1

		npos = strings.IndexByte(r.URL.Path[j:], '/')
		switch npos {
		case -1:
			event = r.URL.Path[j:]
			r.URL.Path = "/"
		case 0:
			// EmptyEvent
			w.WriteHeader(http.StatusBadRequest)
			return
		default:
			event = r.URL.Path[j : npos+j]
			r.URL.Path = r.URL.Path[j+npos:]
		}

		if len(service) == 0 || len(event) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	app, err := cocaine.NewService(ctx, service, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer app.Close()

	task, err := packRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	channel, err := app.Call(ctx, "enqueue", event)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := channel.Call(ctx, "write", task); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var (
		body      []byte
		startLine struct {
			Code    int
			Headers [][2]string
		}
	)

	packedHeaders, err := channel.Get(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
		return
	}
	if err := packedHeaders.ExtractTuple(&body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
		return
	}

	if err := codec.NewDecoderBytes(body, hAsocket).Decode(&startLine); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
		return
	}
	body = body[:]

	log.Println(startLine)
	for _, header := range startLine.Headers {
		w.Header().Add(header[0], header[1])
	}
	w.WriteHeader(startLine.Code)

BODY:
	for {
		res, err := channel.Get(ctx)
		switch {
		case err != nil:
			break BODY
		case res.Err() != nil:
			break BODY
		case channel.Closed():
			break BODY
		default:
			res.ExtractTuple(&body)
			w.Write(body)
			body = body[:]
		}
	}
}

func NewServer() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", process)
	return mux
}
