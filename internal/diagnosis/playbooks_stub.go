//go:build !windows

package diagnosis

import (
	"context"
	"time"
)

func wslExec(_ context.Context, _ string, command string, _ time.Duration) (string, error) {
	return "(stub) " + command, nil
}

func wslRunInDistro(_ context.Context, _, command string, _ time.Duration) (string, error) {
	return "(stub) " + command, nil
}
