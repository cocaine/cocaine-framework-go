package cocaine12

import (
	"bytes"
	"fmt"
	"log"
)

type fallbackLogger struct {
	severity Severity
}

func (f *fallbackLogger) WithFields(fields Fields) *Entry {
	return &Entry{
		Logger: f,
		fields: fields,
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

func (f *fallbackLogger) Log(level Severity, msg string, args []interface{}, fields Fields) {
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
	f.Log(ErrorLevel, format, args, defaultFields)
}

func (f *fallbackLogger) Warnf(format string, args ...interface{}) {
	f.Log(WarnLevel, format, args, defaultFields)
}

func (f *fallbackLogger) Infof(format string, args ...interface{}) {
	f.Log(InfoLevel, format, args, defaultFields)
}

func (f *fallbackLogger) Debugf(format string, args ...interface{}) {
	f.Log(DebugLevel, format, args, defaultFields)
}

func (f *fallbackLogger) Verbosity() Severity {
	return f.severity.get()
}

func (f *fallbackLogger) SetVerbosity(value Severity) {
	f.severity.set(value)
}

func (f *fallbackLogger) Close() {
}
