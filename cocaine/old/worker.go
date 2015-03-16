package cocaine

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/ugorji/go/codec"
)

const (
	HEARTBEAT_TIMEOUT = time.Second * 20
	DISOWN_TIMEOUT    = time.Second * 5
)

type (
	EventHandler func(Request, Response)

	Request interface {
		Read() chan *Message
	}

	Response interface {
		Write(data interface{})
		ErrorMsg(code int, msg string)
		Close()
	}
)

type requestImpl struct {
	from_worker chan *Message
	to_handler  chan *Message
	quit        chan struct{}
}

func newRequest() *requestImpl {
	request := requestImpl{
		from_worker: make(chan *Message),
		to_handler:  make(chan *Message),
		quit:        make(chan struct{}),
	}
	go func() {
		var pending []*Message
		quit := false
		for {
			var out chan *Message
			var first *Message
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

func (request *requestImpl) push(msg *Message) {
	request.from_worker <- msg
}

func (request *requestImpl) close() {
	close(request.quit)
}

func (request *requestImpl) Read() chan *Message {
	return request.to_handler
}

// Datastream from worker to a client.
type responseImpl struct {
	session      uint64
	from_handler chan *Message
	to_worker    chan *Message
	quit         chan struct{}
}

func newResponse(session uint64, to_worker chan *Message) *responseImpl {
	response := responseImpl{
		session:      session,
		from_handler: make(chan *Message),
		to_worker:    to_worker,
		quit:         make(chan struct{}),
	}

	go func() {
		var pending []*Message
		quit := false
		for {
			var out chan *Message
			var first *Message
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
			case <-response.quit:
				quit = true
			}
		}
	}()
	return &response
}

// Sends chunk of data to a client.
func (response *responseImpl) Write(data interface{}) {
	var res []byte
	codec.NewEncoderBytes(&res, h).Encode(&data)
	chunkMsg := Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: response.session,
			MsgType: CHUNK,
		},
		Payload: []interface{}{res},
	}
	response.from_handler <- &chunkMsg
}

// Notify a client about finishing the datastream.
func (response *responseImpl) Close() {
	chokeMsg := Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: response.session,
			MsgType: CHOKE,
		},
		Payload: []interface{}{},
	}
	response.from_handler <- &chokeMsg
	close(response.quit)
}

// Send error to a client. Specify code and message, which describes this error.
func (response *responseImpl) ErrorMsg(code int, msg string) {
	errorMessage := Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: response.session,
			MsgType: ERROR,
		},
		Payload: []interface{}{code, msg},
	}
	response.from_handler <- &errorMessage
}

// Performs IO operations between application
// and cocaine-runtime, dispatches incoming messages from runtime.
type Worker struct {
	socketIO
	uuid            uuid.UUID
	logger          *Logger
	heartbeat_timer *time.Timer
	disown_timer    *time.Timer
	sessions        map[uint64](*requestImpl)
	from_handlers   chan *Message
}

// Creates new instance of Worker. Returns error on fail.
func NewWorker() (worker *Worker, err error) {
	flag.Parse()
	sock, err := newAsyncConnection("unix", flagEndpoint, time.Second*5)
	if err != nil {
		return
	}

	logger, err := NewLogger()
	if err != nil {
		return
	}

	workerID, _ := uuid.FromString(flagUUID)

	w := Worker{
		socketIO:        sock,
		uuid:            workerID,
		logger:          logger,
		heartbeat_timer: time.NewTimer(HEARTBEAT_TIMEOUT),
		disown_timer:    time.NewTimer(DISOWN_TIMEOUT),
		sessions:        make(map[uint64](*requestImpl)),
		from_handlers:   make(chan *Message),
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
		case msg := <-worker.Read():
			switch msg.MsgType {
			case CHUNK:
				worker.logger.Debug("Receive chunk")
				// msg.Payload - change it when patch will be applied to core
				worker.sessions[msg.Session].push(msg)

			case CHOKE:
				worker.logger.Debug("Receive choke")
				worker.sessions[msg.Session].close()
				delete(worker.sessions, msg.Session)

			case INVOKE:
				worker.logger.Debug(fmt.Sprintf("Receive invoke %d", msg.Session))
				cur_session := msg.Session
				req := newRequest()
				resp := newResponse(cur_session, worker.from_handlers)
				worker.sessions[cur_session] = req

				var event string
				event, ok := msg.Payload[0].(string)
				if !ok {
					worker.logger.Err("Corrupted message")
					continue
				}

				if callback, ok := bind[event]; ok {
					go func() {
						defer func() {
							if r := recover(); r != nil {
								errMsg := fmt.Sprintf("Error in event: '%s', exception: %s", event, r)
								worker.logger.Err(fmt.Sprintf("%s \n Stacktrace: \n %s",
									errMsg, string(debug.Stack())))
								resp.ErrorMsg(1, errMsg)
								resp.Close()
							}
						}()
						callback(req, resp)
					}()
				} else {
					errMsg := fmt.Sprintf("There is no event handler for %s", event)
					worker.logger.Debug(errMsg)
					resp.ErrorMsg(-100, errMsg)
					resp.Close()
				}

			case HEARTBEAT:
				worker.logger.Debug("Receive heartbeat. Stop disown_timer")
				worker.disown_timer.Stop()

			case TERMINATE:
				worker.logger.Info("Receive terminate")
				os.Exit(0)

			default:
				worker.logger.Warn("Unknown message")
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
	heartbeat := Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: 0,
			MsgType: HEARTBEAT,
		},
		Payload: []interface{}{},
	}
	worker.Write() <- &heartbeat
	worker.disown_timer.Reset(DISOWN_TIMEOUT)
	worker.heartbeat_timer.Reset(HEARTBEAT_TIMEOUT)
}

func (worker *Worker) handshake() {
	handshake := Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: 0,
			MsgType: HANDSHAKE,
		},
		Payload: []interface{}{worker.uuid},
	}
	worker.Write() <- &handshake
}
