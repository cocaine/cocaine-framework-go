package cocaine12

import (
	"bytes"
	"context"
	"fmt"
	"log"
)

type fallbackLogger struct {
	severity Severity
}

func newFallbackLogger(args ...string) (Logger, error) {
	return &fallbackLogger{
		severity: DebugLevel,
	}, nil
}

func (f *fallbackLogger) WithFields(fields Fields) *Entry {
	return &Entry{
		Logger: f,
		Fields: fields,
	}
}

func (f *fallbackLogger) formatFields(fields Fields) string {
	if len(fields) == 0 {
		return "[ ]"
	}

	var b bytes.Buffer
	b.WriteByte('[')
	b.WriteByte(' ')
	for k, v := range fields {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(fmt.Sprint(v))
		b.WriteByte(' ')
	}
	b.WriteByte(']')

	return b.String()
}

func (f *fallbackLogger) V(level Severity) bool {
	return level >= f.severity.get()
}

func (f *fallbackLogger) log(level Severity, fields Fields, msg string, args ...interface{}) {
	if !f.V(level) {
		return
	}

	if len(fields) == 0 {
		log.Printf("[%s] %s", level.String(), fmt.Sprintf(msg, args...))
	} else {
		log.Printf("[%s] %s %s", level.String(), fmt.Sprintf(msg, args...), f.formatFields(fields))
	}
}

func (f *fallbackLogger) Errf(format string, args ...interface{}) {
	f.log(ErrorLevel, defaultFields, format, args...)
}

func (f *fallbackLogger) Err(args ...interface{}) {
	f.log(ErrorLevel, defaultFields, fmt.Sprint(args...))
}

func (f *fallbackLogger) Warnf(format string, args ...interface{}) {
	f.log(WarnLevel, defaultFields, format, args...)
}

func (f *fallbackLogger) Warn(args ...interface{}) {
	f.log(WarnLevel, defaultFields, fmt.Sprint(args...))
}

func (f *fallbackLogger) Infof(format string, args ...interface{}) {
	f.log(InfoLevel, defaultFields, format, args...)
}

func (f *fallbackLogger) Info(args ...interface{}) {
	f.log(InfoLevel, defaultFields, fmt.Sprint(args...))
}

func (f *fallbackLogger) Debugf(format string, args ...interface{}) {
	f.log(DebugLevel, defaultFields, format, args...)
}

func (f *fallbackLogger) Debug(args ...interface{}) {
	f.log(DebugLevel, defaultFields, fmt.Sprint(args...))
}

func (f *fallbackLogger) Verbosity(context.Context) Severity {
	return f.severity.get()
}

func (f *fallbackLogger) SetVerbosity(value Severity) {
	f.severity.set(value)
}

func (f *fallbackLogger) Close() {
}
