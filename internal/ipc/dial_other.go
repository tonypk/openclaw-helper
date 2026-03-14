//go:build !windows

package ipc

import "net"

func dial(address string) (net.Conn, error) {
	return net.Dial("unix", address)
}
