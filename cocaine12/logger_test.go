package cocaine12

import (
	"testing"
)

func TestLogger(t *testing.T) {
	log, err := NewLogger()
	if err != nil {
		t.Fatal(err)
	}

	// log.SetVerbosity(InfoLevel)
	log.WithFields(Fields{"a": 1, "b": 2}).Errf("test %v", log.Verbosity())
	log.WithFields(Fields{"a": 1, "b": 2}).Warnf("test %v", log.Verbosity())
	log.WithFields(Fields{"a": 1, "b": 2}).Infof("test %v", log.Verbosity())
	log.WithFields(Fields{"a": 1, "b": 2}).Debugf("test %v", log.Verbosity())
}

func BenchmarkFormatFields5(b *testing.B) {
	fields := Fields{
		"A":    1,
		"B":    2,
		"C":    3,
		"TEXT": "TEXT",
	}
	for i := 0; i < b.N; i++ {
		formatFields(fields)
	}
}
