package scriptrun

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultManifestURL is the raw GitHub URL for the manifest.
	DefaultManifestURL = "https://raw.githubusercontent.com/tonypk/openclaw-scripts/main/manifest.json"

	manifestFetchTimeout = 15 * time.Second
)

// Runtime specifies how a script should be executed.
type Runtime string

const (
	RuntimeWSLBash   Runtime = "wsl_bash"
	RuntimePowerShell Runtime = "powershell"
)

// ScriptEntry describes a single script in the manifest.
type ScriptEntry struct {
	URL            string  `json:"url"`             // relative path within repo
	SHA256         string  `json:"sha256"`           // hex-encoded SHA-256 hash
	Runtime        Runtime `json:"runtime"`          // execution runtime
	TimeoutSeconds int     `json:"timeout_seconds"`  // max execution time
	Distro         string  `json:"distro,omitempty"` // WSL distro name (for wsl_bash)
}

// PhaseScripts groups install and verify scripts for a phase.
type PhaseScripts struct {
	Install *ScriptEntry `json:"install,omitempty"`
	Verify  *ScriptEntry `json:"verify,omitempty"`
}

// Manifest is the top-level structure fetched from GitHub.
type Manifest struct {
	Version int                      `json:"version"`
	Phases  map[string]*PhaseScripts `json:"phases"`
}

// FetchManifest downloads and parses the manifest from the given URL.
// Returns the manifest and the ETag for caching. If ifNoneMatch is provided
// and the server returns 304, returns (nil, etag, nil).
func FetchManifest(url string, ifNoneMatch string) (*Manifest, string, error) {
	client := &http.Client{Timeout: manifestFetchTimeout}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	etag := resp.Header.Get("ETag")

	if resp.StatusCode == http.StatusNotModified {
		return nil, etag, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("manifest fetch HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	if err != nil {
		return nil, "", fmt.Errorf("read manifest body: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, "", fmt.Errorf("parse manifest: %w", err)
	}

	return &m, etag, nil
}
