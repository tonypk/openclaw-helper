//go:build windows

package ipc

import (
	"net"

	"github.com/Microsoft/go-winio"
)

func dial(address string) (net.Conn, error) {
	return winio.DialPipe(address, nil)
}
