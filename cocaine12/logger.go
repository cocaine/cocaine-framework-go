package cocaine12

const defaultLoggerName = "logging"

type Fields map[string]interface{}

type EntryLogger interface {
	Errf(format string, args ...interface{})
	Err(args ...interface{})

	Warnf(format string, args ...interface{})
	Warn(args ...interface{})

	Infof(format string, args ...interface{})
	Info(args ...interface{})

	Debugf(format string, args ...interface{})
	Debug(args ...interface{})
}

// Logger represents an interface for a cocaine.Logger
type Logger interface {
	EntryLogger

	log(level Severity, fields Fields, msg string, args ...interface{})
	WithFields(Fields) *Entry

	Verbosity() Severity
	// SetVerbosity(level Severity)
	V(level Severity) bool

	Close()
}

var defaultFields = Fields{}

// NewLogger tries to create a cocaine.Logger. It fallbacks to a simple implementation
// if the cocaine.Logger is unavailable
func NewLogger(endpoints ...string) (Logger, error) {
	return NewLoggerWithName(defaultLoggerName, endpoints...)
}

func NewLoggerWithName(name string, endpoints ...string) (Logger, error) {
	l, err := newCocaineLogger(name, endpoints...)
	if err != nil {
		return newFallbackLogger()
	}
	return l, nil
}
