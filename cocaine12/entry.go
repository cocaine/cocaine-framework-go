package cocaine12

import (
	"fmt"
)

type Entry struct {
	Logger
	Fields Fields
}

// just for the type check
var _ EntryLogger = &Entry{}

func (e *Entry) Errf(format string, args ...interface{}) {
	if e.V(ErrorLevel) {
		e.log(ErrorLevel, e.Fields, format, args...)
	}
}

func (e *Entry) Warnf(format string, args ...interface{}) {
	if e.V(WarnLevel) {
		e.log(WarnLevel, e.Fields, format, args...)
	}
}

func (e *Entry) Infof(format string, args ...interface{}) {
	if e.V(InfoLevel) {
		e.log(InfoLevel, e.Fields, format, args...)
	}
}

func (e *Entry) Debugf(format string, args ...interface{}) {
	if e.V(DebugLevel) {
		e.log(DebugLevel, e.Fields, format, args...)
	}
}

func (e *Entry) Err(args ...interface{}) {
	if e.V(ErrorLevel) {
		e.log(ErrorLevel, e.Fields, fmt.Sprint(args...))
	}
}

func (e *Entry) Warn(args ...interface{}) {
	if e.V(WarnLevel) {
		e.log(WarnLevel, e.Fields, fmt.Sprint(args...))
	}
}

func (e *Entry) Info(args ...interface{}) {
	if e.V(InfoLevel) {
		e.log(InfoLevel, e.Fields, fmt.Sprint(args...))
	}
}

func (e *Entry) Debug(args ...interface{}) {
	if e.V(DebugLevel) {
		e.log(DebugLevel, e.Fields, fmt.Sprint(args...))
	}
}
