//go:build windows

package ipc

import (
	"net"

	"github.com/Microsoft/go-winio"
)

const defaultPipeName = `\\.\pipe\openclaw-helper`

func listen(address string) (net.Listener, error) {
	if address == "" {
		address = defaultPipeName
	}
	return winio.ListenPipe(address, nil)
}
