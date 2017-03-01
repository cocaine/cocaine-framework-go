package cocaine12

import (
	"context"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	ctx, c := context.WithTimeout(context.Background(), time.Second*3)
	defer c()
	log, err := newCocaineLogger(ctx, defaultLoggerName)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	log.WithFields(Fields{"a": 1, "b": 2}).Errf("Error %v", log.Verbosity(ctx))
	log.WithFields(Fields{"a": 1, "b": 2}).Warnf("Warning %v", log.Verbosity(ctx))
	log.WithFields(Fields{"a": 1, "b": 2}).Infof("Info %v", log.Verbosity(ctx))
	log.WithFields(Fields{"a": 1, "b": 2}).Debugf("Debug %v", log.Verbosity(ctx))
}

func BenchmarkFormatMessageWith5Fields(b *testing.B) {
	fields := Fields{
		"A":    1,
		"B":    2,
		"C":    3,
		"TEXT": "TEXT",
	}
	for i := 0; i < b.N; i++ {
		packLogPayload(DebugLevel, "prefix", "workeruuid", fields, "message")
	}
}
