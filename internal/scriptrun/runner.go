package scriptrun

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"time"
)

// RunResult holds the result of a script execution.
type RunResult struct {
	ExitCode     int
	NeedsReboot  bool
	ErrorMessage string
	Diagnostics  map[string]string
	VerifyOK     *bool // nil if no VERIFY message received
}

// ProgressFunc is called for each ##OCH: protocol message parsed from script output.
type ProgressFunc func(msg *ProtocolMessage)

// Runner executes scripts and parses their output in real-time.
type Runner struct {
	cache *Cache
}

// NewRunner creates a new script runner.
func NewRunner(cache *Cache) *Runner {
	return &Runner{cache: cache}
}

// Run executes a script entry and parses its output.
func (r *Runner) Run(ctx context.Context, entry *ScriptEntry, onProgress ProgressFunc) (*RunResult, error) {
	if entry == nil {
		return nil, fmt.Errorf("nil script entry")
	}

	content, err := r.cache.GetScript(entry)
	if err != nil {
		return nil, fmt.Errorf("get script: %w", err)
	}

	timeout := time.Duration(entry.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute // default
	}

	tCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stdout, exitCode, err := executeScript(tCtx, entry.Runtime, entry.Distro, content)
	if tCtx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("script timed out after %v", timeout)
	}

	result := &RunResult{
		ExitCode:    exitCode,
		Diagnostics: make(map[string]string),
	}

	// Parse output lines
	if stdout != nil {
		r.parseOutput(stdout, result, onProgress)
	}

	if err != nil && !result.NeedsReboot {
		if result.ErrorMessage != "" {
			return result, fmt.Errorf("%s", result.ErrorMessage)
		}
		return result, fmt.Errorf("script failed with exit code %d", exitCode)
	}

	return result, nil
}

// parseOutput reads script output line-by-line and processes ##OCH: messages.
func (r *Runner) parseOutput(reader io.Reader, result *RunResult, onProgress ProgressFunc) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[script] %s", line)

		msg := ParseLine(line)
		if msg == nil {
			continue
		}

		switch msg.Type {
		case MsgError:
			result.ErrorMessage = msg.Text
		case MsgReboot:
			result.NeedsReboot = true
		case MsgVerify:
			ok := msg.VerifyOK
			result.VerifyOK = &ok
		case MsgDiag:
			if msg.DiagKey != "" {
				result.Diagnostics[msg.DiagKey] = msg.DiagValue
			}
		}

		if onProgress != nil {
			onProgress(msg)
		}
	}
}
