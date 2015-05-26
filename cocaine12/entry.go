package cocaine12

type Entry struct {
	Logger
	fields Fields
}

// just for the type check
var _ EntryLogger = &Entry{}

func (e *Entry) Errf(format string, args ...interface{}) {
	if e.V(ErrorLevel) {
		e.Log(ErrorLevel, format, args, e.fields)
	}
}

func (e *Entry) Infof(format string, args ...interface{}) {
	if e.V(InfoLevel) {
		e.Log(InfoLevel, format, args, e.fields)
	}
}

func (e *Entry) Warnf(format string, args ...interface{}) {
	if e.V(WarnLevel) {
		e.Log(WarnLevel, format, args, e.fields)
	}
}

func (e *Entry) Debugf(format string, args ...interface{}) {
	if e.V(DebugLevel) {
		e.Log(DebugLevel, format, args, e.fields)
	}
}
