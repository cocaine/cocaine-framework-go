package cocaine12

import (
	"runtime"
)

// dump all stacks
func dumpStack() []byte {
	var (
		buf       []byte
		stackSize int

		bufferLen = 16384
	)

	for stackSize == len(buf) {
		buf = make([]byte, bufferLen)
		stackSize = runtime.Stack(buf, true)
		bufferLen *= 2
	}

	return buf[:stackSize]
}
