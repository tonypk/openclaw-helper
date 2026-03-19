package scriptrun

import (
	"strings"
	"testing"
)

func TestFallbackScriptsList(t *testing.T) {
	fb := NewFallbackScripts()
	paths := fb.List()
	if len(paths) == 0 {
		t.Fatal("no fallback scripts found")
	}

	// Should have scripts for key phases
	found := map[string]bool{
		"node":    false,
		"ubuntu":  false,
		"openclaw": false,
		"config":  false,
		"verify":  false,
	}

	for _, p := range paths {
		for key := range found {
			if strings.Contains(p, key) {
				found[key] = true
			}
		}
	}

	for key, ok := range found {
		if !ok {
			t.Errorf("no fallback script found for phase %q", key)
		}
	}
}

func TestFallbackScriptsGet(t *testing.T) {
	fb := NewFallbackScripts()

	// Test with manifest-style path
	content, ok := fb.Get("scripts/node/install.sh")
	if !ok {
		t.Fatal("expected to find scripts/node/install.sh fallback")
	}
	if !strings.Contains(content, "##OCH:PROGRESS") {
		t.Error("fallback script should contain ##OCH:PROGRESS lines")
	}
	if !strings.Contains(content, "nvm") {
		t.Error("node install script should reference nvm")
	}
}

func TestFallbackScriptsGetMissing(t *testing.T) {
	fb := NewFallbackScripts()

	_, ok := fb.Get("scripts/nonexistent/install.sh")
	if ok {
		t.Error("should not find nonexistent script")
	}
}
