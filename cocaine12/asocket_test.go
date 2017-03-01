package cocaine12

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestASocketDrain(t *testing.T) {
	var exit = make(chan struct{})
	buff := newAsyncBuf()

	var (
		count    = 0
		expected = 3
	)

	msg := &message{}
	for i := 0; i < expected; i++ {
		buff.in <- msg
	}

	go func() {
		buff.Drain(1 * time.Second)
		close(exit)
	}()

	for m := range buff.out {
		count++
		assert.Equal(t, msg, m)
	}

	assert.Equal(t, expected, count)
	<-exit
}

func TestASocketConnect(t *testing.T) {
	_, err := newTCPConnection("128.0.0.1:45000", time.Second)
	assert.Error(t, err)
	_, err = newUnixConnection("unix.sock", time.Second)
	assert.Error(t, err)
}
