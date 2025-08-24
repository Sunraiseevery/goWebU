//go:build windows

package app

import (
	"context"
	"net"
	"syscall"
)

func listenLocal(addr string) (net.Listener, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var err error
			if controlErr := c.Control(func(fd uintptr) {
				err = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			}); controlErr != nil {
				return controlErr
			}
			return err
		},
	}
	return lc.Listen(context.Background(), "tcp", addr)
}
