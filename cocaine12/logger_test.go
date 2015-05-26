package cocaine12

import (
	"testing"
)

func TestLogger(t *testing.T) {
	log, err := NewLogger()
	if err != nil {
		t.Fatal(err)
	}

	log.WithFields(Fields{"a": 1, "b": 2}).Errf("test %v", log.Verbosity())
	log.WithFields(Fields{"a": 1, "b": 2}).Warnf("test %v", log.Verbosity())
	log.WithFields(Fields{"a": 1, "b": 2}).Infof("test %v", log.Verbosity())
	log.WithFields(Fields{"a": 1, "b": 2}).Debugf("test %v", log.Verbosity())
}
