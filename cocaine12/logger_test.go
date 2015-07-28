package cocaine12

import (
	"testing"

	"golang.org/x/net/context"
)

func TestLogger(t *testing.T) {
	log, err := NewLogger()
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	// log.SetVerbosity(InfoLevel)
	log.WithFields(Fields{"a": 1, "b": 2}).Errf("test %v", log.Verbosity(ctx))
	log.WithFields(Fields{"a": 1, "b": 2}).Warnf("test %v", log.Verbosity(ctx))
	log.WithFields(Fields{"a": 1, "b": 2}).Infof("test %v", log.Verbosity(ctx))
	log.WithFields(Fields{"a": 1, "b": 2}).Debugf("test %v", log.Verbosity(ctx))
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
