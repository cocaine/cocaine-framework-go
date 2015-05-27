package cocaine12

const defaultLoggerName = "logging"

type Fields map[string]interface{}

type EntryLogger interface {
	Errorf(format string, args ...interface{})
	Error(format string)

	Warnf(format string, args ...interface{})
	Warn(format string)

	Infof(format string, args ...interface{})
	Info(format string)

	Debugf(format string, args ...interface{})
	Debug(format string)
}

// Logger represents an interface for a cocaine.Logger
type Logger interface {
	EntryLogger

	log(level Severity, fields Fields, msg string, args ...interface{})
	WithFields(Fields) *Entry

	Verbosity() Severity
	SetVerbosity(level Severity)
	V(level Severity) bool

	Close()
}

var defaultFields = Fields{}

// NewLogger tries to create a cocaine.Logger. It fallbacks to a simple implementation
// if the cocaine.Logger is unavailable
func NewLogger(args ...string) (Logger, error) {
	l, err := newCocaineLogger("logging")
	if err != nil {
		return newFallbackLogger()
	}

	return l, err
}
