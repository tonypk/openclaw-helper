package wsl

import (
	"testing"
)

func TestParseDistroList_Normal(t *testing.T) {
	input := `  NAME            STATE           VERSION
* Ubuntu          Running         2
  Debian          Stopped         2
  Alpine          Stopped         1
`
	distros := parseDistroList(input)

	if len(distros) != 3 {
		t.Fatalf("expected 3 distros, got %d", len(distros))
	}

	tests := []struct {
		idx       int
		name      string
		state     string
		version   int
		isDefault bool
	}{
		{0, "Ubuntu", "Running", 2, true},
		{1, "Debian", "Stopped", 2, false},
		{2, "Alpine", "Stopped", 1, false},
	}

	for _, tc := range tests {
		d := distros[tc.idx]
		if d.Name != tc.name {
			t.Errorf("[%d] name: want %q, got %q", tc.idx, tc.name, d.Name)
		}
		if d.State != tc.state {
			t.Errorf("[%d] state: want %q, got %q", tc.idx, tc.state, d.State)
		}
		if d.Version != tc.version {
			t.Errorf("[%d] version: want %d, got %d", tc.idx, tc.version, d.Version)
		}
		if d.IsDefault != tc.isDefault {
			t.Errorf("[%d] isDefault: want %v, got %v", tc.idx, tc.isDefault, d.IsDefault)
		}
	}
}

func TestParseDistroList_Empty(t *testing.T) {
	distros := parseDistroList("")
	if len(distros) != 0 {
		t.Errorf("expected 0 distros, got %d", len(distros))
	}
}

func TestParseDistroList_HeaderOnly(t *testing.T) {
	input := "  NAME            STATE           VERSION\n"
	distros := parseDistroList(input)
	if len(distros) != 0 {
		t.Errorf("expected 0 distros, got %d", len(distros))
	}
}

func TestParseDistroList_WithBOM(t *testing.T) {
	// WSL outputs UTF-16 with BOM and null bytes
	input := "\ufeff  NAME\x00            STATE\x00           VERSION\x00\r\n* Ubuntu\x00          Running\x00         2\x00\r\n"
	distros := parseDistroList(input)

	if len(distros) != 1 {
		t.Fatalf("expected 1 distro, got %d", len(distros))
	}
	if distros[0].Name != "Ubuntu" {
		t.Errorf("name: want Ubuntu, got %q", distros[0].Name)
	}
	if !distros[0].IsDefault {
		t.Error("expected Ubuntu to be default")
	}
}
