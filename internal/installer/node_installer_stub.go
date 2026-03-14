//go:build !windows

package installer

import (
	"context"
	"time"
)

// NodeInstaller is a stub on non-Windows platforms.
type NodeInstaller struct{}

func (n *NodeInstaller) Phase() Phase { return PhaseNode }

func (n *NodeInstaller) Verify() bool { return true }

func (n *NodeInstaller) Execute(_ context.Context, progress func(ProgressEvent)) (bool, error) {
	progress(ProgressEvent{
		Phase: PhaseNode, Status: PhaseCompleted,
		Message: "Node.js check skipped (non-Windows)", Progress: 100,
		Timestamp: time.Now(),
	})
	return false, nil
}
