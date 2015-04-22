package bridge

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
)

type Bridge struct {
	config  *BridgeConfig
	onClose chan struct{}

	worker *cocaine.Worker
	child  *exec.Cmd
}

type BridgeConfig struct {
	Name string
	Args []string
	Env  []string
	Port int
}

func DefaultBridgeConfig() *BridgeConfig {
	name := os.Getenv("NAME")

	cfg := &BridgeConfig{
		Name: name,
		Args: os.Args[1:],
		Env:  os.Environ(),
		Port: 80,
	}
	return cfg
}

// to implement io.Writer
type responseWriter struct {
	w cocaine.Response
}

func (r *responseWriter) Write(body []byte) (int, error) {
	r.w.Write(body)
	return len(body), nil
}

func NewBridge(cfg *BridgeConfig) (*Bridge, error) {
	worker, err := cocaine.NewWorker()
	if err != nil {
		return nil, err
	}

	cli := http.Client{}

	worker.SetFallbackHandler(func(event string, request cocaine.Request, response cocaine.Response) {
		defer response.Close()

		// Read the first chunk
		// It consists of method, uri, httpversion, headers, body.
		// They are packed by msgpack
		msg, err := request.Read()
		if err != nil {
			if cocaine.IsTimeout(err) {
				response.Write(cocaine.WriteHead(http.StatusRequestTimeout, cocaine.Headers{}))
				response.Write("request was not received during a timeout")
				return
			}

			response.Write(cocaine.WriteHead(http.StatusBadRequest, cocaine.Headers{}))
			response.Write("cannot process request " + err.Error())
			return
		}

		httpRequest, err := cocaine.UnpackProxyRequest(msg)
		if err != nil {
			response.Write(cocaine.WriteHead(http.StatusBadRequest, cocaine.Headers{}))
			response.Write("malformed request")
			return
		}

		appResp, err := cli.Do(httpRequest)
		if err != nil {
			response.Write(cocaine.WriteHead(http.StatusInternalServerError, cocaine.Headers{}))
			response.Write("unable to proxy a request")
			return
		}
		defer appResp.Body.Close()

		response.Write(cocaine.WriteHead(appResp.StatusCode, cocaine.HeadersHTTPtoCocaine(appResp.Header)))

		io.Copy(&responseWriter{response}, appResp.Body)
	})

	child := exec.Command(cfg.Name, cfg.Args...)
	// attach environment
	child.Env = cfg.Env
	// the child will be killed if the master died
	child.SysProcAttr = getSysProctAttr()

	b := &Bridge{
		config:  cfg,
		worker:  worker,
		child:   child,
		onClose: make(chan struct{}),
	}

	go b.eventLoop()

	return b, nil
}

func (b *Bridge) Start() {
	go b.worker.Run(nil)
}

func (b *Bridge) eventLoop() {
	// create a watcher
	signalWatcher := make(chan os.Signal, 10)
	signalCHLDWatcher := make(chan os.Signal, 1)
	// attach the signal watcher to watch the signals
	signal.Notify(signalWatcher, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSTOP)
	signal.Notify(signalCHLDWatcher, syscall.SIGCHLD)

	var stopChild = true

	select {
	case <-signalCHLDWatcher:
		// we should stop the worker only
		stopChild = false
	case <-signalWatcher:
		// pass stopChild
	case <-b.onClose:
		// pass to stopChild
	}

	b.stopWorker()

	if stopChild {
		b.stopChild()
	}
}

func (b *Bridge) stopChild() error {
	if err := b.child.Process.Signal(syscall.SIGTERM); err != nil {
		b.child.Process.Kill()
		return err
	}

	// not to stuck in waiting for a child process
	// we will kill it if it takes too long
	notWait := make(chan struct{})
	go func() {
		select {
		case <-time.After(time.Second * 5):
			b.child.Process.Kill()
		case <-notWait:
			return
		}
	}()

	err := b.child.Wait()
	close(notWait)

	return err
}

func (b *Bridge) stopWorker() {
	b.worker.Stop()
}

func (b *Bridge) Close() {
	// notify the eventLoop
	// about an intention to be stopped
	// and wait for an exit of the child process
	close(b.onClose)
	b.waitChild()
}

func (b *Bridge) waitChild() {
	b.child.Process.Wait()
}
