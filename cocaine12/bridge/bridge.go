package bridge

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/context"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
)

type Bridge struct {
	config  *BridgeConfig
	onClose chan struct{}

	worker *cocaine.Worker
	logger cocaine.Logger
	child  *exec.Cmd
}

type BridgeConfig struct {
	Name           string
	Args           []string
	Env            []string
	Port           int
	StartupTimeout int
}

func (cfg *BridgeConfig) Endpoint() string {
	return fmt.Sprintf("localhost:%d", cfg.Port)
}

//Remove some cocaine-specific args
func filterEndpointArg(args []string) []string {
	for i, arg := range args {
		if arg == "--endpoint" && len(args)-1 >= i+1 {
			// cut both the argname and value
			return append(args[:i], args[i+2:]...)
		}
	}

	return args
}

func NewBridgeConfig() *BridgeConfig {
	cfg := &BridgeConfig{
		Name:           "",
		Args:           filterEndpointArg(os.Args[1:]),
		Env:            os.Environ(),
		Port:           8080,
		StartupTimeout: 5,
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

func NewBridge(cfg *BridgeConfig, logger cocaine.Logger) (*Bridge, error) {
	var worker *cocaine.Worker
	worker, err := cocaine.NewWorker()
	if err != nil {
		return nil, err
	}

	endpoint := cfg.Endpoint()

	worker.SetFallbackHandler(func(ctx context.Context, event string, request cocaine.Request, response cocaine.Response) {
		defer response.Close()

		// Read the first chunk
		// It consists of method, uri, httpversion, headers, body.
		// They are packed by msgpack
		msg, err := request.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				response.Write(cocaine.WriteHead(http.StatusRequestTimeout, cocaine.Headers{}))
				response.Write([]byte("request was not received during a timeout"))
				return
			}

			response.Write(cocaine.WriteHead(http.StatusBadRequest, cocaine.Headers{}))
			response.Write([]byte("cannot process request " + err.Error()))
			return
		}

		httpRequest, err := cocaine.UnpackProxyRequest(msg)
		if err != nil {
			response.Write(cocaine.WriteHead(http.StatusBadRequest, cocaine.Headers{}))
			response.Write([]byte(fmt.Sprintf("malformed request: %v", err)))
			return
		}

		// Set scheme and endpoint
		httpRequest.URL.Scheme = "http"
		httpRequest.URL.Host = endpoint

		appResp, err := http.DefaultClient.Do(httpRequest)
		if err != nil {
			response.Write(cocaine.WriteHead(http.StatusInternalServerError, cocaine.Headers{}))
			response.Write([]byte(fmt.Sprintf("unable to proxy a request: %v", err)))
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
		logger:  logger,
		child:   child,
		onClose: make(chan struct{}),
	}

	go b.eventLoop()

	return b, nil
}

func (b *Bridge) Start() error {
	onClose := make(chan struct{})
	defer close(onClose)
	go func() {
		deadline := time.After(time.Duration(b.config.StartupTimeout) * time.Second)
		endpoint := b.config.Endpoint()
	PING_LOOP:
		for {
			if err := ping(endpoint, time.Millisecond*100); err == nil {
				break PING_LOOP
			}

			select {
			case <-deadline:
				return
			case <-onClose:
				return
			default:
			}
		}
		b.worker.Run(nil)
	}()

	stdout, err := b.child.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := b.child.StderrPipe()
	if err != nil {
		return err
	}

	// ToDo: make this goroutines terminatable
	go func() {
		buf := bufio.NewScanner(stdout)
		for buf.Scan() {
			b.logger.Infof("%s", buf.Bytes())
		}

		if err := buf.Err(); err != nil {
			b.logger.Errf("unable to read stdout %v", err)
		}
	}()

	go func() {
		buf := bufio.NewScanner(stderr)
		for buf.Scan() {
			b.logger.Infof("%s", buf.Bytes())
		}

		if err := buf.Err(); err != nil {
			b.logger.Errf("unable to read stderr: %v", err)
		}
	}()

	if err := b.child.Start(); err != nil {
		return err
	}

	return b.child.Wait()
}

func (b *Bridge) eventLoop() {
	// create a watcher
	signalWatcher := make(chan os.Signal, 10)
	signalCHLDWatcher := make(chan os.Signal, 1)
	// attach the signal watcher to watch the signals
	signal.Notify(signalWatcher, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSTOP)
	signal.Notify(signalCHLDWatcher, syscall.SIGCHLD)

	var stopChild = true

	for {
		select {
		case <-signalCHLDWatcher:
			// we should stop the worker only
			stopChild = false
		case <-signalWatcher:
			// pass stopChild
		case <-b.onClose:
			// pass to stopChild
		}
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
