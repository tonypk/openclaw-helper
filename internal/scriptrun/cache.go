package scriptrun

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	scriptFetchTimeout = 30 * time.Second
	// baseRawURL is the base URL for downloading scripts from the repo.
	baseRawURL = "https://raw.githubusercontent.com/tonypk/openclaw-scripts/main/"
)

// cacheState tracks ETag and manifest for persistence.
type cacheState struct {
	ETag     string `json:"etag"`
	Manifest *Manifest `json:"manifest"`
}

// Cache manages downloading, verifying, and caching scripts locally.
type Cache struct {
	mu          sync.RWMutex
	dir         string    // cache directory path
	manifest    *Manifest // current manifest
	etag        string    // last ETag from manifest fetch
	manifestURL string    // URL to fetch manifest from
	fallback    *FallbackScripts
}

// NewCache creates a new script cache.
// dir is the local cache directory (e.g. %APPDATA%/openclaw-helper/scripts/).
// fb is the go:embed fallback scripts (may be nil).
func NewCache(dir string, manifestURL string, fb *FallbackScripts) *Cache {
	if manifestURL == "" {
		manifestURL = DefaultManifestURL
	}
	return &Cache{
		dir:         dir,
		manifestURL: manifestURL,
		fallback:    fb,
	}
}

// CacheDir returns the cache directory path.
func (c *Cache) CacheDir() string {
	return c.dir
}

// Sync fetches the latest manifest and downloads any new/updated scripts.
// Safe to call from a background goroutine.
func (c *Cache) Sync() error {
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	// Load persisted state
	c.loadState()

	c.mu.RLock()
	currentETag := c.etag
	c.mu.RUnlock()

	manifest, etag, err := FetchManifest(c.manifestURL, currentETag)
	if err != nil {
		log.Printf("[scriptcache] manifest fetch failed: %v (using cached/fallback)", err)
		return err
	}

	if manifest == nil {
		// 304 Not Modified
		log.Printf("[scriptcache] manifest unchanged (etag=%s)", etag)
		return nil
	}

	c.mu.Lock()
	c.manifest = manifest
	c.etag = etag
	c.mu.Unlock()

	// Download scripts that are missing or have changed hashes
	var downloadErrors []error
	for phaseName, ps := range manifest.Phases {
		for scriptType, entry := range map[string]*ScriptEntry{"install": ps.Install, "verify": ps.Verify} {
			if entry == nil {
				continue
			}
			localPath := c.scriptPath(entry.URL)
			if c.hashMatches(localPath, entry.SHA256) {
				continue
			}
			log.Printf("[scriptcache] downloading %s/%s: %s", phaseName, scriptType, entry.URL)
			if err := c.downloadScript(entry.URL, entry.SHA256); err != nil {
				log.Printf("[scriptcache] download failed for %s: %v", entry.URL, err)
				downloadErrors = append(downloadErrors, err)
			}
		}
	}

	// Persist state
	c.saveState()

	if len(downloadErrors) > 0 {
		return fmt.Errorf("%d script download(s) failed", len(downloadErrors))
	}

	if err := c.SyncResources(); err != nil {
		log.Printf("[cache] resource sync failed: %v", err)
	}
	return nil
}

// GetScript returns the content of a script for the given entry.
// It checks: 1) cached file with valid hash, 2) fallback embedded script.
func (c *Cache) GetScript(entry *ScriptEntry) (string, error) {
	if entry == nil {
		return "", fmt.Errorf("nil script entry")
	}

	// Try cached file
	localPath := c.scriptPath(entry.URL)
	if c.hashMatches(localPath, entry.SHA256) {
		data, err := os.ReadFile(localPath)
		if err == nil {
			return string(data), nil
		}
	}

	// Try fallback
	if c.fallback != nil {
		if content, ok := c.fallback.Get(entry.URL); ok {
			log.Printf("[scriptcache] using fallback for %s", entry.URL)
			return content, nil
		}
	}

	return "", fmt.Errorf("script not available: %s (not cached and no fallback)", entry.URL)
}

// GetManifest returns the current manifest (thread-safe).
func (c *Cache) GetManifest() *Manifest {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.manifest
}

// scriptPath returns the local filesystem path for a script URL.
func (c *Cache) scriptPath(url string) string {
	return filepath.Join(c.dir, filepath.FromSlash(url))
}

// hashMatches checks if the file at path has the expected SHA-256 hash.
func (c *Cache) hashMatches(path, expectedHash string) bool {
	if expectedHash == "" {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	actual := hex.EncodeToString(h.Sum(nil))
	return actual == expectedHash
}

// downloadScript downloads a script and verifies its hash.
func (c *Cache) downloadScript(relURL, expectedHash string) error {
	fullURL := baseRawURL + relURL

	client := &http.Client{Timeout: scriptFetchTimeout}
	resp, err := client.Get(fullURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", relURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", relURL, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB max
	if err != nil {
		return fmt.Errorf("read %s: %w", relURL, err)
	}

	// Verify hash
	h := sha256.Sum256(body)
	actual := hex.EncodeToString(h[:])
	if actual != expectedHash {
		return fmt.Errorf("hash mismatch for %s: expected %s, got %s", relURL, expectedHash, actual)
	}

	// Write to cache
	localPath := c.scriptPath(relURL)
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("create dir for %s: %w", relURL, err)
	}
	if err := os.WriteFile(localPath, body, 0644); err != nil {
		return fmt.Errorf("write %s: %w", relURL, err)
	}

	return nil
}

func (c *Cache) stateFilePath() string {
	return filepath.Join(c.dir, ".cache-state.json")
}

func (c *Cache) saveState() {
	c.mu.RLock()
	state := cacheState{
		ETag:     c.etag,
		Manifest: c.manifest,
	}
	c.mu.RUnlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.Printf("[scriptcache] failed to marshal state: %v", err)
		return
	}
	if err := os.WriteFile(c.stateFilePath(), data, 0644); err != nil {
		log.Printf("[scriptcache] failed to save state: %v", err)
	}
}

func (c *Cache) loadState() {
	data, err := os.ReadFile(c.stateFilePath())
	if err != nil {
		return
	}
	var state cacheState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("[scriptcache] failed to parse cached state: %v", err)
		return
	}
	c.mu.Lock()
	c.etag = state.ETag
	if state.Manifest != nil {
		c.manifest = state.Manifest
	}
	c.mu.Unlock()
}

// SyncResources downloads playbooks, diagnostics, FAQ, and config if manifest V2.
func (c *Cache) SyncResources() error {
	c.mu.RLock()
	m := c.manifest
	c.mu.RUnlock()

	if m == nil {
		return nil
	}

	resources := map[string]*ResourceEntry{
		"playbooks":   m.Playbooks,
		"diagnostics": m.Diagnostics,
		"faq":         m.FAQ,
		"config":      m.Config,
	}

	for name, res := range resources {
		if res == nil {
			continue
		}
		localPath := filepath.Join(c.dir, name+".json")
		if c.hashMatches(localPath, res.SHA256) {
			continue
		}
		if err := c.downloadResourceTo(res.URL, res.SHA256, localPath); err != nil {
			log.Printf("[cache] failed to download %s: %v", name, err)
		}
	}

	// Download repair scripts
	if m.RepairScripts != nil {
		for _, entry := range m.RepairScripts {
			localPath := filepath.Join(c.dir, entry.URL)
			if c.hashMatches(localPath, entry.SHA256) {
				continue
			}
			dir := filepath.Dir(localPath)
			os.MkdirAll(dir, 0755)
			if err := c.downloadResourceTo(entry.URL, entry.SHA256, localPath); err != nil {
				log.Printf("[cache] failed to download repair script %s: %v", entry.URL, err)
			}
		}
	}

	return nil
}

// downloadResourceTo downloads a resource from the remote base URL and saves to a specific local path.
func (c *Cache) downloadResourceTo(relURL, expectedHash, localPath string) error {
	fullURL := baseRawURL + relURL
	client := &http.Client{Timeout: scriptFetchTimeout}
	resp, err := client.Get(fullURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, fullURL)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return err
	}

	// Verify hash
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != expectedHash {
		return fmt.Errorf("hash mismatch for %s", relURL)
	}

	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(localPath, data, 0644)
}

// GetResource returns the content of a cached resource file (playbooks.json, diagnostics.json, etc.).
func (c *Cache) GetResource(name string) ([]byte, error) {
	localPath := filepath.Join(c.dir, name+".json")
	return os.ReadFile(localPath)
}

// GetRepairScript returns the content of a cached repair script by relative path.
func (c *Cache) GetRepairScript(relativePath string) ([]byte, error) {
	localPath := filepath.Join(c.dir, relativePath)
	return os.ReadFile(localPath)
}

// ForceSync re-fetches the manifest and all resources, ignoring ETag cache.
func (c *Cache) ForceSync() error {
	c.mu.Lock()
	c.etag = ""
	c.mu.Unlock()
	if err := c.Sync(); err != nil {
		return err
	}
	return c.SyncResources()
}
