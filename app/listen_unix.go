//go:build !windows

package app

import "net"

func listenLocal(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}
