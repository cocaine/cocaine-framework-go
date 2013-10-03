package cocaine

import (
	"codec"
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	flag_uuid     string
	flag_endpoint string
	flag_app      string
	flag_locator  string
)

func init() {
	flag.StringVar(&flag_uuid, "uuid", "", "UUID")
	flag.StringVar(&flag_endpoint, "endpoint", "", "Connection path")
	flag.StringVar(&flag_app, "app", "", "Connection path")
	flag.StringVar(&flag_locator, "locator", "", "Connection path")
	flag.Parse()
}

const (
	HEARTBEAT_TIMEOUT = time.Second * 20
	DISOWN_TIMEOUT    = time.Second * 5
)

//Request
type Request struct {
	from_worker chan []byte
	to_handler  chan []byte
	quit        chan bool
}

type EventHandler func(*Request, *Response)

func NewRequest() *Request {
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
				pending = pending[1:]
			case <-request.quit:
				quit = true
			}
		}
	}()
	return &request
}

func (request *Request) push(data []byte) {
	go func() {
		request.from_worker <- data
	}()
}

func (request *Request) close() {
	request.quit <- true
}

func (request *Request) Read() chan []byte {
	return request.to_handler
}

//Response
type Response struct {
	session      int64
	from_handler chan []byte
	to_worker    chan RawMessage
}

func NewResponse(session int64, to_worker chan RawMessage) *Response {
	response := Response{session, make(chan []byte), to_worker}
	go func() {
		var pending [][]byte
		for {
			var out chan RawMessage
			var first RawMessage
			if len(pending) > 0 {
				first = pending[0]
				out = to_worker
			}
			select {
			case incoming := <-response.from_handler:
				pending = append(pending, incoming)
			case out <- first:
				pending = pending[1:]
			}
		}
	}()
	return &response
}

func (response *Response) Write(data interface{}) {
	var res []byte
	codec.NewEncoderBytes(&res, h).Encode(&data)
	response.from_handler <- Pack(&Chunk{MessageInfo{CHUNK, response.session}, res})
}

func (response *Response) Close() {
	response.from_handler <- Pack(&Choke{MessageInfo{CHOKE, response.session}})
}

//Worker
type Worker struct {
	pipe            *Pipe
	unpacker        *StreamUnpacker
	uuid            string
	wr_in           chan RawMessage
	r_out           chan RawMessage
	logger          *Logger
	heartbeat_timer *time.Timer
	disown_timer    *time.Timer
	sessions        map[int64](*Request)
	from_handlers   chan RawMessage
}

func NewWorker() *Worker {
	wr_in, wr_out := Transmite() // Write to buffer: wr_in <-
	r_in, r_out := Transmite()   // Read from buffer: <- r_out
	pipe := NewPipe("unix", flag_endpoint, &wr_out, &r_in)
	w := Worker{pipe: pipe,
		unpacker:        NewStreamUnpacker(),
		uuid:            flag_uuid,
		wr_in:           wr_in,
		r_out:           r_out,
		logger:          NewLogger(),
		heartbeat_timer: time.NewTimer(HEARTBEAT_TIMEOUT),
		disown_timer:    time.NewTimer(DISOWN_TIMEOUT),
		sessions:        make(map[int64](*Request)),
		from_handlers:   make(chan RawMessage)}
	w.disown_timer.Stop()
	w.handshake()
	w.heartbeat()
	return &w
}

func (worker *Worker) Loop(bind map[string]EventHandler) {
	for {
		select {
		case answer := <-worker.r_out:
			msgs := worker.unpacker.Feed(answer)
			for _, rawmsg := range msgs {
				switch msg := rawmsg.(type) {
				case *Chunk:
					worker.logger.Err("Receive chunk")
					worker.sessions[msg.GetSessionID()].push(msg.Data)

				case *Choke:
					worker.logger.Info("Receive choke")
					worker.sessions[msg.GetSessionID()].close()
					delete(worker.sessions, msg.GetSessionID())

				case *Invoke:
					worker.logger.Info(fmt.Sprintf("Receive invoke %s %d", msg.Event, msg.GetSessionID()))
					cur_session := msg.GetSessionID()
					req := NewRequest()
					resp := NewResponse(cur_session, worker.from_handlers)
					worker.sessions[cur_session] = req
					if callback, ok := bind[msg.Event]; ok {
						go callback(req, resp)
					} else {
						worker.logger.Info(fmt.Sprintf("There is no event handler for %s", msg.Event))
					}

				case *Heartbeat:
					worker.logger.Info("Receive heartbeat. Stop disown_timer")
					worker.disown_timer.Stop()

				case *Terminate:
					worker.logger.Info("Receive terminate")
					os.Exit(0)

				default:
					worker.logger.Info("Unknown message")
				}
			}
		case <-worker.heartbeat_timer.C:
			worker.logger.Info("Send heartbeat")
			worker.heartbeat()

		case <-worker.disown_timer.C:
			worker.logger.Err("Disowned")

		case outcoming := <-worker.from_handlers:
			worker.wr_in <- outcoming
		}
	}
}

func (worker *Worker) heartbeat() {
	heartbeat := Heartbeat{MessageInfo{HEARTBEAT, 0}}
	worker.wr_in <- Pack(&heartbeat)
	worker.disown_timer.Reset(DISOWN_TIMEOUT)
	worker.heartbeat_timer.Reset(HEARTBEAT_TIMEOUT)
}

func (worker *Worker) handshake() {
	handshake := Handshake{MessageInfo{HANDSHAKE, 0}, worker.uuid}
	worker.wr_in <- Pack(&handshake)
}
