package cocaine12

import (
	"context"
	"fmt"
	"sync"

	"github.com/tinylib/msgp/msgp"
)

const (
	loggerEmit      = 0
	uuidLoggerField = "uuid"
)

type cocaineLogger struct {
	*Service

	mu         sync.Mutex
	severity   Severity
	prefix     string
	workerUuid string
}

func newCocaineLogger(ctx context.Context, name string, endpoints ...string) (Logger, error) {
	service, err := NewService(ctx, name, endpoints)
	if err != nil {
		return nil, err
	}

	logger := &cocaineLogger{
		Service:    service,
		severity:   -100,
		prefix:     fmt.Sprintf("app/%s", GetDefaults().ApplicationName()),
		workerUuid: GetDefaults().UUID(),
	}

	return logger, nil
}

func (c *cocaineLogger) Close() {
	c.Service.Close()
}

func (c *cocaineLogger) Verbosity(ctx context.Context) (level Severity) {
	level = DebugLevel
	if lvl := c.severity.get(); lvl != -100 {
		return lvl
	}

	channel, err := c.Service.Call(ctx, "verbosity")
	if err != nil {
		return
	}

	result, err := channel.Get(ctx)
	if err != nil {
		return
	}

	var verbosity struct {
		Level Severity
	}

	err = result.Extract(&verbosity)
	if err != nil {
		return
	}

	c.severity.set(verbosity.Level)

	return verbosity.Level
}

func (c *cocaineLogger) V(level Severity) bool {
	return level >= c.severity.get()
}

func (c *cocaineLogger) WithFields(fields Fields) *Entry {
	return &Entry{
		Logger: c,
		Fields: fields,
	}
}

func packPair(b []byte, key string, value interface{}) []byte {
	b = msgp.AppendArrayHeader(b, 2)
	b = msgp.AppendString(b, key)
	b, err := msgp.AppendIntf(b, value)
	if err != nil {
		b = msgp.AppendString(b, err.Error())
	}
	return b
}

func packLogPayload(level Severity, prefix string, u string, fields Fields, msg string, args ...interface{}) []byte {
	var formattedMessage = msg
	if len(args) > 0 {
		formattedMessage = fmt.Sprintf(msg, args...)
	}
	// pack message args
	// 4 = level + prefix + message + tags
	const sz = 4
	// TODO: calculate capacity for fields
	var b = make([]byte, 0, msgp.Int32Size+msgp.StringPrefixSize+len(formattedMessage))
	// Payload array header
	b = msgp.AppendArrayHeader(b, sz)
	// log level
	b = msgp.AppendInt32(b, int32(level))
	// logger prefix
	b = msgp.AppendString(b, prefix)
	// log message
	b = msgp.AppendString(b, formattedMessage)
	// pack fields
	// NOTE: I hope we will never overflow uin32
	if u != "" {
		b = msgp.AppendArrayHeader(b, uint32(len(fields))+1)
		b = packPair(b, uuidLoggerField, u)
	} else {
		b = msgp.AppendArrayHeader(b, uint32(len(fields)))
	}

	for k, v := range fields {
		b = packPair(b, k, v)
	}
	return b
}

func (c *cocaineLogger) log(level Severity, fields Fields, msg string, args ...interface{}) {
	payload := packLogPayload(level, c.prefix, c.workerUuid, fields, msg, args)
	c.mu.Lock()
	defer c.mu.Unlock()
	loggermsg := newMessageRawArgs(c.Service.sessions.Next(), loggerEmit, msgp.Raw(payload), nil)
	c.Service.sendMsg(loggermsg)
}

func (c *cocaineLogger) Debug(args ...interface{}) {
	if c.V(DebugLevel) {
		c.log(DebugLevel, defaultFields, fmt.Sprint(args...))
	}
}

func (c *cocaineLogger) Debugf(msg string, args ...interface{}) {
	if c.V(DebugLevel) {
		c.log(DebugLevel, defaultFields, msg, args...)
	}
}

func (c *cocaineLogger) Info(args ...interface{}) {
	if c.V(InfoLevel) {
		c.log(InfoLevel, defaultFields, fmt.Sprint(args...))
	}
}

func (c *cocaineLogger) Infof(msg string, args ...interface{}) {
	if c.V(InfoLevel) {
		c.log(InfoLevel, defaultFields, msg, args...)
	}
}

func (c *cocaineLogger) Warn(args ...interface{}) {
	if c.V(WarnLevel) {
		c.log(WarnLevel, defaultFields, fmt.Sprint(args...))
	}
}

func (c *cocaineLogger) Warnf(msg string, args ...interface{}) {
	if c.V(WarnLevel) {
		c.log(WarnLevel, defaultFields, msg, args...)
	}
}

func (c *cocaineLogger) Err(args ...interface{}) {
	if c.V(ErrorLevel) {
		c.log(ErrorLevel, defaultFields, fmt.Sprint(args...))
	}
}

func (c *cocaineLogger) Errf(msg string, args ...interface{}) {
	if c.V(ErrorLevel) {
		c.log(ErrorLevel, defaultFields, msg, args...)
	}
}
