// +build darwin

package bridge

import (
	"syscall"
)

func getSysProctAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
