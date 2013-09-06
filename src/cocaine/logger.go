package cocaine

import (
	"fmt"
)

type Logger struct {
	host  string
	port  uint64
	pipe  *Pipe
	wr_in chan RawMessage
	r_out chan RawMessage
}

const (
	_ = iota
	LOGERROR
	LOGWARN
	LOGINFO
	LOGDEBUG
)

func NewLogger() *Logger {
	l := NewLocator("localhost", 10053)
	info := <-l.Resolve("logging")
	wr_in, wr_out := Transmite()
	r_in, r_out := Transmite()
	pipe := NewPipe("tcp", info.Endpoint.AsString(), &wr_out, &r_in)
	return &Logger{info.Endpoint.Host, info.Endpoint.Port, pipe, wr_in, r_out}
}

func (logger *Logger) log(level int64, message string) {
	msg := ServiceMethod{MessageInfo{0, 0}, []interface{}{level, fmt.Sprintf("app/%s", flag_app), message}}
	logger.wr_in <- Pack(&msg)
}

func (logger *Logger) Err(message string) {
	go logger.log(LOGERROR, message)
}

func (logger *Logger) Warn(message string) {
	go logger.log(LOGWARN, message)
}

func (logger *Logger) Info(message string) {
	go logger.log(LOGINFO, message)
}

func (logger *Logger) Debug(message string) {
	go logger.log(LOGDEBUG, message)
}
