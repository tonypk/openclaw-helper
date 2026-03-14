//go:build !windows

package ipc

import (
	"net"
	"os"
	"path/filepath"
)

func listen(address string) (net.Listener, error) {
	if address == "" {
		address = filepath.Join(os.TempDir(), "openclaw-helper.sock")
	}
	// Remove stale socket file
	os.Remove(address)
	return net.Listen("unix", address)
}
