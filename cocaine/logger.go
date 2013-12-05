package cocaine

import (
	"errors"
	"fmt"
	"time"
)

type Logger struct {
	socketWriter
	verbosity int
}

const (
	LOGINGNORE = iota
	LOGERROR
	LOGWARN
	LOGINFO
	LOGDEBUG
)

func NewLogger(args ...interface{}) (logger *Logger, err error) {
	temp, err := NewService("logging", args...)
	if err != nil {
		return
	}
	defer temp.Close()

	res := <-temp.Call("verbosity")
	if res.Err() != nil {
		err = errors.New("cocaine: unable to receive verbosity")
		return
	}
	verbosity := 0
	if err = res.Extract(&verbosity); err != nil {
		return
	}

	sock, err := newWSocket("tcp", temp.ResolveResult.AsString(), time.Second*5)
	if err != nil {
		return
	}

	//Create logger
	logger = &Logger{sock, verbosity}
	return
}

// Blocked
func (logger *Logger) log(level int64, message ...interface{}) bool {
	msg := ServiceMethod{messageInfo{0, 0}, []interface{}{level, fmt.Sprintf("app/%s", flagApp), fmt.Sprint(message...)}}
	logger.Write() <- packMsg(&msg)
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
