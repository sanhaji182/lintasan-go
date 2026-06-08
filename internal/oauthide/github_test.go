package oauthide

import "testing"

func TestGitHubDeviceStartShape(t *testing.T) {
	// Integration test skipped in CI — shape only
	if GitHubClientID == "" || GitHubDeviceCodeURL == "" {
		t.Fatal("github constants missing")
	}
}