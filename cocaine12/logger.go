package cocaine12

const defaultLoggerName = "logging"

type Fields map[string]interface{}

type EntryLogger interface {
	Errf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

// Logger represents an interface for a cocaine.Logger
type Logger interface {
	EntryLogger

	Log(level Severity, msg string, args []interface{}, fields Fields)
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
	return &fallbackLogger{}, nil
}
