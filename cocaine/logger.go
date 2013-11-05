package cocaine

import (
	"fmt"
	"github.com/ugorji/go/codec"
	"runtime/debug"
	"time"
)

type Logger struct {
	socketIO
	stop      chan bool
	verbosity int
}

const (
	LOGINGNORE = iota
	LOGERROR
	LOGWARN
	LOGINFO
	LOGDEBUG
)

func NewLogger() *Logger {
	l, _ := NewLocator("localhost", 10053)
	defer l.Close()
	info := <-l.Resolve("logging")
	if info.success == false {
		fmt.Printf("Unable to create logger, could not resolve logging service. Stack trace: \n %s", string(debug.Stack()))
		return nil
	}
	sock, err := NewASocket("tcp", info.Endpoint.AsString(), time.Second*5)
	if err != nil {
		return nil
	}

	//Create logger
	logger := Logger{
		sock, make(chan bool), 0}

	//Get verbosity
	msg := ServiceMethod{MessageInfo{1, 0}, []interface{}{}}
	logger.socketIO.Write() <- Pack(&msg)
	u := NewStreamUnpacker()
	for {
		for _, item := range u.Feed(<-logger.socketIO.Read()) {
			switch msg := item.(type) {
			case *Chunk:
				codec.NewDecoderBytes(msg.Data, h).Decode(&logger.verbosity)
			case *Choke:
				return &logger
			}
		}
	}
}

// Blocked
func (logger *Logger) log(level int64, message string) bool {
	msg := ServiceMethod{MessageInfo{0, 0}, []interface{}{level, fmt.Sprintf("app/%s", flag_app), message}}
	logger.socketIO.Write() <- Pack(&msg)
	<-logger.socketIO.Read() // Blocked

	return true
}

func (logger *Logger) Err(message string) {
	_ = LOGERROR <= logger.verbosity && logger.log(LOGERROR, message)
}

func (logger *Logger) Warn(message string) {
	_ = LOGWARN <= logger.verbosity && logger.log(LOGWARN, message)
}

func (logger *Logger) Info(message string) {
	_ = LOGINFO <= logger.verbosity && logger.log(LOGINFO, message)
}

func (logger *Logger) Debug(message string) {
	_ = LOGDEBUG <= logger.verbosity && logger.log(LOGDEBUG, message)
}

func (logger *Logger) Close() {
	logger.socketIO.Close()
	select {
	case logger.stop <- true:
	default:
		return
	}
}
