package cocaine

import (
	"fmt"
	"sync"
	"time"
)

type Logger struct {
	socketWriter
	verbosity       int
	args            []interface{}
	mutex           sync.Mutex
	is_reconnecting bool
}

const (
	LOGINGNORE = iota
	LOGERROR
	LOGWARN
	LOGINFO
	LOGDEBUG
)

func createTempService(args ...interface{}) (endpoint string, verbosity int, err error) {
	temp, err := NewService("logging", args...)
	if err != nil {
		return
	}
	defer temp.Close()

	res := <-temp.Call("verbosity")
	if res.Err() != nil {
		err = fmt.Errorf("%s", "cocaine: unable to receive verbosity")
		return
	}
	verbosity = 0
	if err = res.Extract(&verbosity); err != nil {
		return
	}
	endpoint = temp.ResolveResult.AsString()
	return
}

func createIO(args ...interface{}) (sock socketWriter, verbosity int, err error) {
	endpoint, verbosity, err := createTempService(args...)
	if err != nil {
		return
	}
	sock, err = newWSocket("tcp", endpoint, time.Second*5)
	if err != nil {
		return
	}
	return
}

func NewLogger(args ...interface{}) (logger *Logger, err error) {
	sock, verbosity, err := createIO(args...)
	if err != nil {
		return
	}

	//Create logger
	logger = &Logger{sock, verbosity, args, sync.Mutex{}, false}
	return
}

func (logger *Logger) Reconnect(force bool) error {
	if !logger.is_reconnecting { // double check
		logger.mutex.Lock()
		defer logger.mutex.Unlock()

		if logger.is_reconnecting {
			return fmt.Errorf("%s", "Service is reconnecting now")
		}
		logger.is_reconnecting = true
		defer func() { logger.is_reconnecting = false }()

		if !force {
			select {
			case <-logger.IsClosed():
			default:
				return fmt.Errorf("%s", "Service is already connected")
			}
		}

		// Create new socket
		sock, verbosity, err := createIO(logger.args...)
		if err != nil {
			return err
		}

		// Dispose old IO interface
		logger.socketWriter.Close()
		// Reset IO interface
		logger.socketWriter = sock
		// Reset verbosity
		logger.verbosity = verbosity
		return nil
	}
	return fmt.Errorf("%s", "Service is reconnecting now")
}

func (logger *Logger) log(level int64, message ...interface{}) bool {
	for {
		select {
		case <-logger.IsClosed():
			err := logger.Reconnect(false)
			if err != nil {
				return false
			}
		default:
			msg := ServiceMethod{messageInfo{0, 0}, []interface{}{level, fmt.Sprintf("app/%s", flagApp), fmt.Sprint(message...)}}
			logger.Write() <- packMsg(&msg)
			return true
		}
	}
	return true
}

func (logger *Logger) Err(message ...interface{}) {
	_ = LOGERROR <= logger.verbosity && logger.log(LOGERROR, message...)
}

func (logger *Logger) Errf(format string, args ...interface{}) {
	_ = LOGERROR <= logger.verbosity && logger.log(LOGERROR, fmt.Sprintf(format, args...))
}

func (logger *Logger) Warn(message ...interface{}) {
	_ = LOGWARN <= logger.verbosity && logger.log(LOGWARN, message...)
}

func (logger *Logger) Warnf(format string, args ...interface{}) {
	_ = LOGWARN <= logger.verbosity && logger.log(LOGWARN, fmt.Sprintf(format, args...))
}

func (logger *Logger) Info(message ...interface{}) {
	_ = LOGINFO <= logger.verbosity && logger.log(LOGINFO, message...)
}

func (logger *Logger) Infof(format string, args ...interface{}) {
	_ = LOGINFO <= logger.verbosity && logger.log(LOGINFO, fmt.Sprintf(format, args...))
}

func (logger *Logger) Debug(message ...interface{}) {
	_ = LOGDEBUG <= logger.verbosity && logger.log(LOGDEBUG, message...)
}

func (logger *Logger) Debugf(format string, args ...interface{}) {
	_ = LOGDEBUG <= logger.verbosity && logger.log(LOGDEBUG, fmt.Sprintf(format, args...))
}

func (logger *Logger) Close() {
	logger.socketWriter.Close()
}
