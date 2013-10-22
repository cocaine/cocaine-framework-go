package cocaine

import (
	"fmt"
	"runtime/debug"
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
	if info.success == false {
		fmt.Printf("Unable to create logger, could not resolve logging service. Stack trace: \n %s", string(debug.Stack()))
		return nil
	}
	wr_in, wr_out := Transmite()
	r_in, r_out := Transmite()
	pipe := NewPipe("tcp", info.Endpoint.AsString(), &wr_out, &r_in)
	return &Logger{info.Endpoint.Host, info.Endpoint.Port, pipe, wr_in, r_out}
}

func (logger *Logger) log(level int64, message string) {
	msg := ServiceMethod{MessageInfo{0, 0}, []interface{}{level, fmt.Sprintf("app/%s", flag_app), message}}
	logger.wr_in <- Pack(&msg)
	<-logger.r_out
}

func (logger *Logger) Err(message string) {
	//logger.log(LOGERROR, message)
}

func (logger *Logger) Warn(message string) {
	//logger.log(LOGWARN, message)
}

func (logger *Logger) Info(message string) {
	//logger.log(LOGINFO, message)
}

func (logger *Logger) Debug(message string) {
	//logger.log(LOGDEBUG, message)
}
