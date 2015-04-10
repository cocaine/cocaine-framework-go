package cocaine

import (
	"fmt"
	"log"
)

const (
	LOGDEBUG = iota
	LOGINFO
	LOGWARN
	LOGERROR
)

const (
	defaultLoggerName = "logging"
)

type Logger interface {
	Err(message ...interface{})
	Errf(format string, args ...interface{})

	Warn(message ...interface{})
	Warnf(format string, args ...interface{})

	Info(message ...interface{})
	Infof(format string, args ...interface{})

	Debug(message ...interface{})
	Debugf(format string, args ...interface{})

	Verbosity() int
	SetVerbosity(level int)

	Close()
}

type fallbackLogger struct{}

func (f *fallbackLogger) Err(message ...interface{}) {
	log.Printf("[ERROR] %s", message...)
}

func (f *fallbackLogger) Errf(format string, args ...interface{}) {
	log.Printf("[ERROR] %s", fmt.Sprintf(format, args...))
}

func (f *fallbackLogger) Warn(message ...interface{}) {
	log.Printf("[WARN] %s", message...)
}

func (f *fallbackLogger) Warnf(format string, args ...interface{}) {
	log.Printf("[WARN] %s", fmt.Sprintf(format, args...))
}

func (f *fallbackLogger) Info(message ...interface{}) {
	log.Printf("[INFO] %s", message...)
}

func (f *fallbackLogger) Infof(format string, args ...interface{}) {
	log.Printf("[INFO] %s", fmt.Sprintf(format, args...))
}

func (f *fallbackLogger) Debug(message ...interface{}) {
	log.Printf("[DEBUG] %s", message...)
}

func (f *fallbackLogger) Debugf(format string, args ...interface{}) {
	log.Printf("[DEBUG] %s", fmt.Sprintf(format, args...))
}

func (f *fallbackLogger) Verbosity() int {
	return LOGDEBUG
}

func (f *fallbackLogger) SetVerbosity(level int) {
}

func (f *fallbackLogger) Close() {
}

func NewLogger(args ...string) (Logger, error) {
	return &fallbackLogger{}, nil
}
