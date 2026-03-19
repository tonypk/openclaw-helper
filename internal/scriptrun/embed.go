package scriptrun

import (
	"embed"
	"io/fs"
	"strings"
)

//go:embed all:fallback
var fallbackFS embed.FS

// FallbackScripts provides access to embedded fallback scripts.
type FallbackScripts struct {
	fs embed.FS
}

// NewFallbackScripts creates a FallbackScripts from the embedded filesystem.
func NewFallbackScripts() *FallbackScripts {
	return &FallbackScripts{fs: fallbackFS}
}

// Get returns the content of a fallback script by its relative path
// (matching the manifest URL format, e.g. "scripts/node/install.sh").
func (f *FallbackScripts) Get(relPath string) (string, bool) {
	// Manifest URLs are like "scripts/node/install.sh".
	// Embedded FS has files at "fallback/node/install.sh".
	// Convert: "scripts/X" → "fallback/X"
	embeddedPath := relPath
	if strings.HasPrefix(relPath, "scripts/") {
		embeddedPath = "fallback/" + strings.TrimPrefix(relPath, "scripts/")
	}

	data, err := fs.ReadFile(f.fs, embeddedPath)
	if err == nil {
		return string(data), true
	}

	// Also try the path as-is (in case it already uses the fallback prefix)
	data, err = fs.ReadFile(f.fs, relPath)
	if err == nil {
		return string(data), true
	}

	return "", false
}

// List returns all available fallback script paths.
func (f *FallbackScripts) List() []string {
	var paths []string
	fs.WalkDir(f.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	return paths
}
