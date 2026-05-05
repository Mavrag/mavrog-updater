package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Repository constants.
const (
	AddonRepoOwner = "Mavrag"
	AddonRepoName  = "MavrogBattleCry-Release"
	// NOTE: zip asset names start with "MavrogBattlecry" (lowercase c).
	// pickAddonAsset matches case-insensitively so this is fine.

	// Updater self-update repo. Change if you host the updater elsewhere.
	UpdaterRepoOwner = "Mavrag"
	UpdaterRepoName  = "mavrog-updater"
)

// ReleaseAsset matches relevant fields from the GitHub Releases API.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Release represents a GitHub release (subset).
type Release struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	Draft       bool           `json:"draft"`
	Prerelease  bool           `json:"prerelease"`
	HTMLURL     string         `json:"html_url"`
	PublishedAt time.Time      `json:"published_at"`
	Assets      []ReleaseAsset `json:"assets"`
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

func ghGet(url string, out interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "MavrogUpdater")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github api %s: %s: %s", url, resp.Status, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// fetchLatestRelease returns the latest non-draft release for the given repo.
// Falls back to listing releases if /latest 404s (e.g. only prereleases).
func fetchLatestRelease(owner, repo string) (*Release, error) {
	var r Release
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	err := ghGet(url, &r)
	if err == nil && r.TagName != "" {
		return &r, nil
	}
	var list []Release
	url2 := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=10", owner, repo)
	if err2 := ghGet(url2, &list); err2 != nil {
		if err != nil {
			return nil, err
		}
		return nil, err2
	}
	for _, rel := range list {
		if rel.Draft {
			continue
		}
		r := rel
		return &r, nil
	}
	return nil, fmt.Errorf("no releases published for %s/%s", owner, repo)
}

// pickAddonAsset picks the best zip asset from a release.
// Strategy:
//  1. Among .zip assets whose name starts with the addon folder name,
//     return the LARGEST (guards against tiny stray uploads / CI artefacts).
//  2. Otherwise, return the largest .zip asset.
//  3. Otherwise, nil.
func pickAddonAsset(r *Release) *ReleaseAsset {
	if r == nil {
		return nil
	}
	prefix := strings.ToLower(AddonFolderName)
	var best, fallback *ReleaseAsset
	for i := range r.Assets {
		a := &r.Assets[i]
		lower := strings.ToLower(a.Name)
		if !strings.HasSuffix(lower, ".zip") {
			continue
		}
		// Skip GitHub's auto-generated source archives (named after the repo).
		if strings.HasPrefix(lower, strings.ToLower(AddonRepoName)) {
			continue
		}
		if strings.HasPrefix(lower, prefix) {
			if best == nil || a.Size > best.Size {
				best = a
			}
			continue
		}
		if fallback == nil || a.Size > fallback.Size {
			fallback = a
		}
	}
	if best != nil {
		return best
	}
	return fallback
}
