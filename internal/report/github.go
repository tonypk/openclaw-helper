package report

import (
	"fmt"
	"net/url"
)

const (
	repoOwner = "tonypk"
	repoName  = "openclaw-helper"
	// Maximum URL length to stay within browser limits.
	maxURLLength = 8000
)

// BuildIssueURL constructs a GitHub new-issue URL with pre-filled content.
func BuildIssueURL(r CrashReport) string {
	body := FormatGitHubBody(r)

	// Truncate body if URL would be too long
	baseURL := fmt.Sprintf("https://github.com/%s/%s/issues/new", repoOwner, repoName)

	params := url.Values{}
	params.Set("title", r.Title)
	params.Set("labels", "crash-report")
	params.Set("body", body)

	fullURL := baseURL + "?" + params.Encode()

	// If too long, truncate the body and retry
	if len(fullURL) > maxURLLength {
		// Estimate how much body we can keep
		overhead := len(baseURL) + len("?title=") + len(url.QueryEscape(r.Title)) +
			len("&labels=crash-report&body=")
		maxBody := maxURLLength - overhead
		if maxBody < 200 {
			maxBody = 200
		}

		// Truncate body (account for URL encoding expansion ~3x)
		truncLen := maxBody / 3
		if truncLen > len(body) {
			truncLen = len(body)
		}
		truncated := body[:truncLen] + "\n\n---\n*Report truncated due to URL length limit. Full report sent to Telegram.*"

		params.Set("body", truncated)
		fullURL = baseURL + "?" + params.Encode()
	}

	return fullURL
}
