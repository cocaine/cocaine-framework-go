package cocaine12

import (
	"strconv"
	"sync/atomic"
)

type Severity int32

const (
	DebugLevel Severity = iota
	InfoLevel
	WarnLevel
	ErrorLevel = 3
)

func (s *Severity) String() string {
	switch i := s.get(); i {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARNING"
	case ErrorLevel:
		return "ERROR"
	default:
		return strconv.FormatInt(int64(i), 10)
	}
}

func (s *Severity) get() Severity {
	return Severity(atomic.LoadInt32((*int32)(s)))
}

func (s *Severity) set(value Severity) {
	atomic.StoreInt32((*int32)(s), int32(value))
}
