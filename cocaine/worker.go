package cocaine

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/satori/go.uuid"
	"github.com/ugorji/go/codec"
)

var (
	flagUUID     string
	flagEndpoint string
	flagApp      string
	flagLocator  string
)

func init() {
	flag.StringVar(&flagUUID, "uuid", "", "UUID")
	flag.StringVar(&flagEndpoint, "endpoint", "", "Connection path")
	flag.StringVar(&flagApp, "app", "standalone", "Connection path")
	flag.StringVar(&flagLocator, "locator", "", "Connection path")
	flag.Parse()
}

const (
	HEARTBEAT_TIMEOUT = time.Second * 20
	DISOWN_TIMEOUT    = time.Second * 5
)

type Request struct {
	from_worker chan []byte
	to_handler  chan []byte
	quit        chan bool
}

type EventHandler func(*Request, *Response)

func newRequest() *Request {
	request := Request{make(chan []byte), make(chan []byte), make(chan bool)}
	go func() {
		var pending [][]byte
		quit := false
		for {
			var out chan []byte
			var first []byte
			if len(pending) > 0 {
				first = pending[0]
				out = request.to_handler
			} else {
				if quit {
					return
				}
			}
			select {
			case incoming := <-request.from_worker:
				pending = append(pending, incoming)
			case out <- first:
				pending[0] = nil
				pending = pending[1:]
			case <-request.quit:
				quit = true
			}
		}
	}()
	return &request
}

func (request *Request) push(data []byte) {
	request.from_worker <- data
}

func (request *Request) close() {
	request.quit <- true
}

func (request *Request) Read() chan []byte {
	return request.to_handler
}

// Datastream from worker to a client.
type Response struct {
	session      int64
	from_handler chan []byte
	to_worker    chan rawMessage
	quit         chan bool
}

func newResponse(session int64, to_worker chan rawMessage) *Response {
	response := Response{session, make(chan []byte), to_worker, make(chan bool)}
	go func() {
		var pending [][]byte
		quit := false
		for {
			var out chan rawMessage
			var first rawMessage
			if len(pending) > 0 {
				first = pending[0]
				out = to_worker
			} else {
				if quit {
					return
				}
			}
			select {
			case incoming := <-response.from_handler:
				pending = append(pending, incoming)
			case out <- first:
				pending[0] = nil
				pending = pending[1:]
			case quit = <-response.quit:
				quit = true
			}
		}
	}()
	return &response
}

// Sends chunk of data to a client.
func (response *Response) Write(data interface{}) {
	var res []byte
	codec.NewEncoderBytes(&res, h).Encode(&data)
	response.from_handler <- packMsg(&chunk{messageInfo{CHUNK, response.session}, res})
}

// Notify a client about finishing the datastream.
func (response *Response) Close() {
	response.from_handler <- packMsg(&choke{messageInfo{CHOKE, response.session}})
	response.quit <- true
}

// Send error to a client. Specify code and message, which describes this error.
func (response *Response) ErrorMsg(code int, msg string) {
	response.from_handler <- packMsg(&errorMsg{messageInfo{ERROR, response.session}, code, msg})
}

// Performs IO operations between application
// and cocaine-runtime, dispatches incoming messages from runtime.
type Worker struct {
	unpacker        *streamUnpacker
	uuid            uuid.UUID
	logger          *Logger
	heartbeat_timer *time.Timer
	disown_timer    *time.Timer
	sessions        map[int64](*Request)
	from_handlers   chan rawMessage
	socketIO
}

// Creates new instance of Worker. Returns error on fail.
func NewWorker() (worker *Worker, err error) {
	sock, err := newAsyncRWSocket("unix", flagEndpoint, time.Second*5)
	if err != nil {
		return
	}

	logger, err := NewLogger()
	if err != nil {
		return
	}

	workerID, _ := uuid.FromString(flagUUID)

	w := Worker{
		unpacker:        newStreamUnpacker(),
		uuid:            workerID,
		logger:          logger,
		heartbeat_timer: time.NewTimer(HEARTBEAT_TIMEOUT),
		disown_timer:    time.NewTimer(DISOWN_TIMEOUT),
		sessions:        make(map[int64](*Request)),
		from_handlers:   make(chan rawMessage),
		socketIO:        sock,
	}
	w.disown_timer.Stop()
	w.handshake()
	w.heartbeat()
	worker = &w
	return
}

// Initializes worker in runtime as starting. Launchs an eventloop.
func (worker *Worker) Loop(bind map[string]EventHandler) {
	for {
		select {
		case answer := <-worker.Read():
			msgs := worker.unpacker.Feed(answer)
			for _, rawmsg := range msgs {
				switch msg := rawmsg.(type) {
				case *chunk:
					worker.logger.Debug("Receive chunk")
					worker.sessions[msg.getSessionID()].push(msg.Data)

				case *choke:
					worker.logger.Debug("Receive choke")
					worker.sessions[msg.getSessionID()].close()
					delete(worker.sessions, msg.getSessionID())

				case *invoke:
					worker.logger.Debug(fmt.Sprintf("Receive invoke %s %d", msg.Event, msg.getSessionID()))
					cur_session := msg.getSessionID()
					req := newRequest()
					resp := newResponse(cur_session, worker.from_handlers)
					worker.sessions[cur_session] = req
					if callback, ok := bind[msg.Event]; ok {
						go func() {
							defer func() {
								if r := recover(); r != nil {
									errMsg := fmt.Sprintf("Error in event: '%s', exception: %s", msg.Event, r)
									worker.logger.Err(fmt.Sprintf("%s \n Stacktrace: \n %s",
										errMsg, string(debug.Stack())))
									resp.ErrorMsg(1, errMsg)
									resp.Close()
								}
							}()
							callback(req, resp)
						}()
					} else {
						errMsg := fmt.Sprintf("There is no event handler for %s", msg.Event)
						worker.logger.Debug(errMsg)
						resp.ErrorMsg(-100, errMsg)
						resp.Close()
					}

				case *heartbeat:
					worker.logger.Debug("Receive heartbeat. Stop disown_timer")
					worker.disown_timer.Stop()

				case *terminateStruct:
					worker.logger.Info("Receive terminate")
					os.Exit(0)

				default:
					worker.logger.Warn("Unknown message")
				}
			}
		case <-worker.heartbeat_timer.C:
			worker.logger.Debug("Send heartbeat")
			worker.heartbeat()

		case <-worker.disown_timer.C:
			worker.logger.Info("Disowned")
			os.Exit(0)

		case outcoming := <-worker.from_handlers:
			worker.Write() <- outcoming
		}
	}
}

func (worker *Worker) heartbeat() {
	heartbeat := heartbeat{messageInfo{HEARTBEAT, 0}}
	worker.Write() <- packMsg(&heartbeat)
	worker.disown_timer.Reset(DISOWN_TIMEOUT)
	worker.heartbeat_timer.Reset(HEARTBEAT_TIMEOUT)
}

func (worker *Worker) handshake() {
	handshake := handshakeStruct{messageInfo{HANDSHAKE, 0}, worker.uuid}
	worker.Write() <- packMsg(&handshake)
}
