//go:build windows

package installer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tonypk/openclaw-helper/pkg/wsl"
)

// NodeInstaller installs Node.js 22+ inside WSL via nvm.
type NodeInstaller struct{}

func (n *NodeInstaller) Phase() Phase { return PhaseNode }

func (n *NodeInstaller) Verify() bool {
	stdout, _, err := wsl.RunWithTimeout("Ubuntu", "node -v", 10*time.Second)
	if err != nil {
		return false
	}
	version := strings.TrimSpace(strings.TrimPrefix(stdout, "v"))
	parts := strings.SplitN(version, ".", 3)
	if len(parts) == 0 {
		return false
	}
	major, err := strconv.Atoi(parts[0])
	return err == nil && major >= 22
}

func (n *NodeInstaller) Execute(ctx context.Context, progress func(ProgressEvent)) (bool, error) {
	if n.Verify() {
		return false, nil
	}

	progress(ProgressEvent{
		Phase: PhaseNode, Status: PhaseRunning,
		Message: "Installing nvm (Node Version Manager)...", Progress: 10,
	})

	// Install nvm
	nvmInstall := `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash`
	_, _, err := wsl.RunWithTimeout("Ubuntu", nvmInstall, 3*time.Minute)
	if err != nil {
		return false, fmt.Errorf("nvm install failed: %w", err)
	}

	progress(ProgressEvent{
		Phase: PhaseNode, Status: PhaseRunning,
		Message: "Installing Node.js 22 LTS...", Progress: 40,
	})

	// Install Node.js 22 via nvm
	nodeInstall := `export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nvm install 22 && nvm alias default 22`
	_, _, err = wsl.RunWithTimeout("Ubuntu", nodeInstall, 5*time.Minute)
	if err != nil {
		return false, fmt.Errorf("Node.js install failed: %w", err)
	}

	progress(ProgressEvent{
		Phase: PhaseNode, Status: PhaseRunning,
		Message: "Verifying Node.js installation...", Progress: 80,
	})

	// Verify
	if !n.Verify() {
		return false, fmt.Errorf("Node.js installation could not be verified")
	}

	progress(ProgressEvent{
		Phase: PhaseNode, Status: PhaseCompleted,
		Message: "Node.js 22 installed successfully", Progress: 100,
	})
	return false, nil
}
