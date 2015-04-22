// +build linux

package bridge

import (
	"syscall"
)

func getSysProctAttr() *syscall.SysProcAttr {
	attrs := &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}

	return attrs
}
