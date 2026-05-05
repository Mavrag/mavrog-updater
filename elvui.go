package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ElvUIFolderName = "ElvUI"
	ElvUIAPIURL     = "https://api.tukui.org/v1/addon/elvui"
)

type ElvUIRelease struct {
	Version    string `json:"version"`
	URL        string `json:"url"`
	Changelog  string `json:"changelog"`
	WebURL     string `json:"web_url"`
}

type ElvUIInfo struct {
	Installed        bool   `json:"installed"`
	InstalledVersion string `json:"installedVersion"`
	LatestVersion    string `json:"latestVersion"`
	DownloadURL      string `json:"downloadUrl"`
	Changelog        string `json:"changelog"`
	WebURL           string `json:"webUrl"`
	UpdateAvailable  bool   `json:"updateAvailable"`
}

func isElvUIInstalled(addonsPath string) bool {
	if addonsPath == "" {
		return false
	}
	st, err := os.Stat(filepath.Join(addonsPath, ElvUIFolderName))
	return err == nil && st.IsDir()
}

func readElvUIVersion(addonsPath string) string {
	if addonsPath == "" {
		return ""
	}
	dir := filepath.Join(addonsPath, ElvUIFolderName)
	names := []string{"ElvUI.toc", "ElvUI_Mainline.toc", "ElvUI-Mainline.toc"}
	for _, n := range names {
		if v, err := readTocVersion(filepath.Join(dir, n)); err == nil && v != "" {
			return v
		}
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".toc") {
			if v, err := readTocVersion(filepath.Join(dir, e.Name())); err == nil && v != "" {
				return v
			}
		}
	}
	return ""
}

func fetchElvUIRelease() (*ElvUIRelease, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", ElvUIAPIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MavrogUpdater")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tukui api: %s", resp.Status)
	}
	var r ElvUIRelease
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return &r, nil
}
