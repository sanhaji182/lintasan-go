package oauthide

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// GitHub Copilot OAuth (ported from 9router GITHUB_CONFIG).
const (
	GitHubClientID        = "Iv1.b507a08c87ecfe98"
	GitHubDeviceCodeURL  = "https://github.com/login/device/code"
	GitHubTokenURL       = "https://github.com/login/oauth/access_token"
	GitHubUserInfoURL    = "https://api.github.com/user"
	GitHubCopilotTokenURL = "https://api.github.com/copilot_internal/v2/token"
	GitHubAPIVersion     = "2022-11-28"
	GitHubUserAgent      = "GitHubCopilotChat/0.26.7"
	GitHubScope          = "read:user"
)

// DeviceStart is the response from starting a device code flow.
type DeviceStart struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// StartGitHubDevice requests a GitHub device code (9router github.requestDeviceCode).
func StartGitHubDevice() (*DeviceStart, error) {
	form := url.Values{}
	form.Set("client_id", GitHubClientID)
	form.Set("scope", GitHubScope)

	req, err := http.NewRequest(http.MethodPost, GitHubDeviceCodeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github device code HTTP %d: %s", resp.StatusCode, truncateErr(string(body)))
	}
	var out DeviceStart
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	return &out, nil
}

// DevicePollResult from one poll attempt.
type DevicePollResult struct {
	Done         bool
	Pending      bool
	Error        string
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	FlowMeta     string // JSON: copilot + user info
}

// PollGitHubDeviceOnce polls token endpoint once (9router github.pollToken).
func PollGitHubDeviceOnce(deviceCode string) (*DevicePollResult, error) {
	form := url.Values{}
	form.Set("client_id", GitHubClientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequest(http.MethodPost, GitHubTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var data map[string]any
	_ = json.Unmarshal(body, &data)

	if errCode, _ := data["error"].(string); errCode == "authorization_pending" {
		return &DevicePollResult{Pending: true}, nil
	}
	if errCode, _ := data["error"].(string); errCode == "slow_down" {
		return &DevicePollResult{Pending: true, Error: "slow_down"}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github poll HTTP %d: %s", resp.StatusCode, truncateErr(string(body)))
	}

	access, _ := data["access_token"].(string)
	refresh, _ := data["refresh_token"].(string)
	expiresIn := 0
	if v, ok := data["expires_in"].(float64); ok {
		expiresIn = int(v)
	}
	if access == "" {
		desc, _ := data["error_description"].(string)
		if desc == "" {
			desc, _ = data["error"].(string)
		}
		return nil, fmt.Errorf("github poll: %s", desc)
	}

	meta, err := githubPostExchange(access)
	if err != nil {
		return nil, err
	}
	metaJSON, _ := json.Marshal(meta)

	return &DevicePollResult{
		Done:         true,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    expiresIn,
		FlowMeta:     string(metaJSON),
	}, nil
}

func githubPostExchange(githubAccess string) (map[string]any, error) {
	meta := map[string]any{}
	ghHeaders := http.Header{
		"Authorization":       []string{"Bearer " + githubAccess},
		"Accept":              []string{"application/json"},
		"X-GitHub-Api-Version": []string{GitHubAPIVersion},
		"User-Agent":          []string{GitHubUserAgent},
	}

	copilotReq, _ := http.NewRequest(http.MethodGet, GitHubCopilotTokenURL, nil)
	copilotReq.Header = ghHeaders.Clone()
	copilotRes, err := httpClient.Do(copilotReq)
	if err == nil {
		defer copilotRes.Body.Close()
		if copilotRes.StatusCode >= 200 && copilotRes.StatusCode < 300 {
			var copilot map[string]any
			if json.NewDecoder(io.LimitReader(copilotRes.Body, 1<<20)).Decode(&copilot) == nil {
				meta["copilot"] = copilot
			}
		}
	}

	userReq, _ := http.NewRequest(http.MethodGet, GitHubUserInfoURL, nil)
	userReq.Header = ghHeaders.Clone()
	userRes, err := httpClient.Do(userReq)
	if err == nil {
		defer userRes.Body.Close()
		if userRes.StatusCode >= 200 && userRes.StatusCode < 300 {
			var user map[string]any
			if json.NewDecoder(io.LimitReader(userRes.Body, 1<<20)).Decode(&user) == nil {
				meta["github_user"] = user
			}
		}
	}
	return meta, nil
}