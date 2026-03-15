package report

import (
	"strings"
	"testing"
)

func TestBuildIssueURL(t *testing.T) {
	r := sampleReport()
	u := BuildIssueURL(r)

	// Check base URL
	if !strings.HasPrefix(u, "https://github.com/tonypk/openclaw-helper/issues/new?") {
		t.Errorf("unexpected URL prefix: %s", u)
	}

	// Check title param
	if !strings.Contains(u, "title=") {
		t.Error("URL missing title param")
	}

	// Check labels
	if !strings.Contains(u, "labels=crash-report") {
		t.Error("URL missing crash-report label")
	}

	// Check body param
	if !strings.Contains(u, "body=") {
		t.Error("URL missing body param")
	}

	// Check within length limit
	if len(u) > maxURLLength+500 {
		t.Errorf("URL too long: %d bytes (max %d)", len(u), maxURLLength)
	}
}

func TestBuildIssueURL_Truncation(t *testing.T) {
	r := sampleReport()
	// Add very long description to trigger truncation
	r.Description = strings.Repeat("This is a very long error description. ", 500)

	u := BuildIssueURL(r)

	if len(u) > maxURLLength+500 {
		t.Errorf("URL should be truncated but is %d bytes", len(u))
	}
}
