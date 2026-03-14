package wsl

import (
	"os/exec"
	"strconv"
	"strings"
)

// Distro represents a WSL distribution.
type Distro struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	Version   int    `json:"version"`
	IsDefault bool   `json:"is_default"`
}

// ListDistros returns all installed WSL distributions.
func ListDistros() ([]Distro, error) {
	cmd := exec.Command("wsl.exe", "--list", "--verbose")
	hideWindow(cmd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return parseDistroList(string(out)), nil
}

// IsInstalled checks if a specific distribution is installed.
func IsInstalled(name string) bool {
	distros, err := ListDistros()
	if err != nil {
		return false
	}
	lower := strings.ToLower(name)
	for _, d := range distros {
		if strings.ToLower(d.Name) == lower {
			return true
		}
	}
	return false
}

// GetWSLVersion returns the default WSL version (1 or 2).
func GetWSLVersion() (int, error) {
	cmd := exec.Command("wsl.exe", "--status")
	hideWindow(cmd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	output := string(out)
	// Look for default version info
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "2") && (strings.Contains(strings.ToLower(line), "default") || strings.Contains(strings.ToLower(line), "version")) {
			return 2, nil
		}
	}

	// If we got output but couldn't determine version, try listing distros
	distros, err := ListDistros()
	if err != nil {
		return 0, err
	}
	for _, d := range distros {
		if d.Version == 2 {
			return 2, nil
		}
	}
	if len(distros) > 0 {
		return distros[0].Version, nil
	}

	return 1, nil
}

// parseDistroList parses the output of `wsl --list --verbose`.
// Example output:
//
//	NAME            STATE          VERSION
//	* Ubuntu        Running        2
//	  Debian        Stopped        2
func parseDistroList(output string) []Distro {
	var distros []Distro
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// Remove BOM and null bytes (wsl.exe outputs UTF-16)
		line = strings.Map(func(r rune) rune {
			if r == '\x00' || r == '\ufeff' || r == '\r' {
				return -1
			}
			return r
		}, line)
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "NAME") {
			continue
		}

		isDefault := strings.HasPrefix(line, "*")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		ver, _ := strconv.Atoi(fields[len(fields)-1])
		state := fields[len(fields)-2]
		name := strings.Join(fields[:len(fields)-2], " ")

		distros = append(distros, Distro{
			Name:      name,
			State:     state,
			Version:   ver,
			IsDefault: isDefault,
		})
	}

	return distros
}
