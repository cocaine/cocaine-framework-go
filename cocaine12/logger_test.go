package cocaine12

import (
	"testing"

	"golang.org/x/net/context"
)

func TestLogger(t *testing.T) {
	ctx := context.Background()
	log, err := NewLogger(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	log.WithFields(Fields{"a": 1, "b": 2}).Errf("Error %v", log.Verbosity(ctx))
	log.WithFields(Fields{"a": 1, "b": 2}).Warnf("Warning %v", log.Verbosity(ctx))
	log.WithFields(Fields{"a": 1, "b": 2}).Infof("Info %v", log.Verbosity(ctx))
	log.WithFields(Fields{"a": 1, "b": 2}).Debugf("Debug %v", log.Verbosity(ctx))
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
