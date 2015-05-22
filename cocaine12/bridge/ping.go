package bridge

import (
	"net"
	"time"
)

func ping(address string, timeout time.Duration) error {
	d := net.Dialer{
		Timeout:   timeout,
		DualStack: true,
	}

	conn, err := d.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()

	return nil
}
