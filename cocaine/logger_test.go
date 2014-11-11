package cocaine

import (
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	logger, err := NewLogger()
	t.Log(logger)
	if err != nil {
		t.Fatalf("unable to create logger %s", err)
	}

	t.Logf("Verbosity %d", logger.Verbosity())
	logger.Debug("DEBUG")
	logger.Info("INFO")
	logger.Err("ERROR")

	time.Sleep(1 * time.Second)
}
