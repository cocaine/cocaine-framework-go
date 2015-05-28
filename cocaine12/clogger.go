package cocaine12

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
)

const loggerEmit = 0

type cocaineLogger struct {
	*Service
	severity Severity
}

func newCocaineLogger(name string) (Logger, error) {
	timeout := time.Second * 5
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	service, err := NewService(ctx, name, nil)
	if err != nil {
		return nil, err
	}

	logger := &cocaineLogger{
		Service:  service,
		severity: -100,
	}
	return logger, nil
}

func (c *cocaineLogger) Close() {
	c.Service.Close()
}

func (c *cocaineLogger) SetVerbosity(level Severity) {
	c.Service.Call("set_verbosity", level)
	c.severity.set(-100)
}

func (c *cocaineLogger) Verbosity() (level Severity) {
	level = DebugLevel
	if lvl := c.severity.get(); lvl != -100 {
		return lvl
	}

	channel, err := c.Service.Call("verbosity")
	if err != nil {
		return
	}

	result, err := channel.Get()
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

func (c *cocaineLogger) formatFields(f Fields) []interface{} {
	formatted := make([]interface{}, 0, len(f))
	for k, v := range f {
		formatted = append(formatted, []interface{}{k, v})
	}

	return formatted
}

func (c *cocaineLogger) WithFields(fields Fields) *Entry {
	return &Entry{
		Logger: c,
		Fields: fields,
	}
}

func (c *cocaineLogger) log(level Severity, fields Fields, msg string, args ...interface{}) {
	var methodArgs []interface{}
	if len(args) > 0 {
		methodArgs = []interface{}{level, defaultAppName, fmt.Sprintf(msg, args...), c.formatFields(fields)}
	} else {
		methodArgs = []interface{}{level, defaultAppName, msg, c.formatFields(fields)}
	}

	i := c.Service.sessions.Next()

	loggermsg := &Message{
		CommonMessageInfo{
			i,
			loggerEmit},
		methodArgs,
	}

	c.Service.sendMsg(loggermsg)
}

func (c *cocaineLogger) Debug(args ...interface{}) {
	c.log(DebugLevel, defaultFields, fmt.Sprint(args...))
}

func (c *cocaineLogger) Debugf(msg string, args ...interface{}) {
	c.log(DebugLevel, defaultFields, msg, args...)
}

func (c *cocaineLogger) Info(args ...interface{}) {
	c.log(InfoLevel, defaultFields, fmt.Sprint(args...))
}

func (c *cocaineLogger) Infof(msg string, args ...interface{}) {
	c.log(InfoLevel, defaultFields, msg, args...)
}

func (c *cocaineLogger) Warn(args ...interface{}) {
	c.log(WarnLevel, defaultFields, fmt.Sprint(args...))
}

func (c *cocaineLogger) Warnf(msg string, args ...interface{}) {
	c.log(WarnLevel, defaultFields, msg, args...)
}

func (c *cocaineLogger) Err(args ...interface{}) {
	c.log(ErrorLevel, defaultFields, fmt.Sprint(args...))
}

func (c *cocaineLogger) Errf(msg string, args ...interface{}) {
	c.log(ErrorLevel, defaultFields, msg, args...)
}
