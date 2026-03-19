package scriptrun

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchManifest(t *testing.T) {
	manifest := Manifest{
		Version: 1,
		Phases: map[string]*PhaseScripts{
			"node": {
				Install: &ScriptEntry{
					URL:            "scripts/node/install.sh",
					SHA256:         "abc123",
					Runtime:        RuntimeWSLBash,
					TimeoutSeconds: 300,
					Distro:         "Ubuntu",
				},
			},
		},
	}
	data, _ := json.Marshal(manifest)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		etag := r.Header.Get("If-None-Match")
		if etag == `"test-etag"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"test-etag"`)
		w.Write(data)
	}))
	defer srv.Close()

	// First fetch
	m, etag, err := FetchManifest(srv.URL, "")
	if err != nil {
		t.Fatalf("FetchManifest: %v", err)
	}
	if m == nil {
		t.Fatal("manifest should not be nil")
	}
	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}
	if etag != `"test-etag"` {
		t.Errorf("ETag = %q", etag)
	}

	// Check phase
	ps, ok := m.Phases["node"]
	if !ok {
		t.Fatal("missing node phase")
	}
	if ps.Install == nil {
		t.Fatal("missing node install script")
	}
	if ps.Install.Runtime != RuntimeWSLBash {
		t.Errorf("Runtime = %s", ps.Install.Runtime)
	}

	// Second fetch with ETag (should get 304)
	m2, _, err := FetchManifest(srv.URL, `"test-etag"`)
	if err != nil {
		t.Fatalf("FetchManifest 304: %v", err)
	}
	if m2 != nil {
		t.Error("manifest should be nil for 304")
	}
}

func TestFetchManifestBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, _, err := FetchManifest(srv.URL, "")
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestFetchManifest404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, _, err := FetchManifest(srv.URL, "")
	if err == nil {
		t.Error("expected error for 404")
	}
}
