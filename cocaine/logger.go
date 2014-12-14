package cocaine

import (
	"fmt"
	"sync"
)

//{0: ['emit', {}, {}],
// 1: ['verbosity', {}, {0: ['value', {}], 1: ['error', {}]}],
// 2: ['set_verbosity', {}, {0: ['value', {}], 1: ['error', {}]}]}

type Logger struct {
	mutex sync.Mutex
	*Service
	verbosity       int
	args            []string
	is_reconnecting bool
	name            string
	target          string
}

const (
	LOGDEBUG = iota
	LOGINFO
	LOGWARN
	LOGERROR
)

func NewLogger(args ...string) (logger *Logger, err error) {
	return NewLoggerWithName("logging", args...)
}

func NewLoggerWithName(loggerName string, args ...string) (logger *Logger, err error) {
	service, err := NewService(loggerName, args...)
	if err != nil {
		return
	}

	resCh, err := service.Call("verbosity")
	if err != nil {
		err = fmt.Errorf("cocaine: unable to receive verbosity")
		return
	}

	res, err := resCh.Get()
	if res.Err() != nil {
		err = fmt.Errorf("cocaine: unable to receive verbosity: %s", res.Err())
	}

	var verbosity struct {
		Verbosity int
	}

	if err := res.Extract(&verbosity); err != nil {
		return nil, fmt.Errorf("cocaine: unable to extract verbosity: %s", err)
	}

	logger = &Logger{
		Service:         service,
		verbosity:       verbosity.Verbosity,
		args:            args,
		is_reconnecting: false,
		name:            loggerName,
		target:          fmt.Sprintf("app/%s", DefaultAppName),
	}
	return
}

func (logger *Logger) Reconnect(force bool) error {
	return logger.Service.Reconnect(force)
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
			logger.Service.Call("emit", level, logger.target, fmt.Sprint(message...))
			return true
		}
	}
	return true
}

func (logger *Logger) Err(message ...interface{}) {
	_ = LOGERROR >= logger.verbosity && logger.log(LOGERROR, message...)
}

func (logger *Logger) Errf(format string, args ...interface{}) {
	_ = LOGERROR >= logger.verbosity && logger.log(LOGERROR, fmt.Sprintf(format, args...))
}

func (logger *Logger) Warn(message ...interface{}) {
	_ = LOGWARN >= logger.verbosity && logger.log(LOGWARN, message...)
}

func (logger *Logger) Warnf(format string, args ...interface{}) {
	_ = LOGWARN >= logger.verbosity && logger.log(LOGWARN, fmt.Sprintf(format, args...))
}

func (logger *Logger) Info(message ...interface{}) {
	_ = LOGINFO >= logger.verbosity && logger.log(LOGINFO, message...)
}

func (logger *Logger) Infof(format string, args ...interface{}) {
	_ = LOGINFO >= logger.verbosity && logger.log(LOGINFO, fmt.Sprintf(format, args...))
}

func (logger *Logger) Debug(message ...interface{}) {
	_ = LOGDEBUG >= logger.verbosity && logger.log(LOGDEBUG, message...)
}

func (logger *Logger) Debugf(format string, args ...interface{}) {
	_ = LOGDEBUG >= logger.verbosity && logger.log(LOGDEBUG, fmt.Sprintf(format, args...))
}

func (logger *Logger) Close() {
	logger.socketIO.Close()
}

func (l *Logger) Verbosity() int {
	return l.verbosity
}
