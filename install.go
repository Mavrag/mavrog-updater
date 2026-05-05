package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// downloadFile downloads url to dest, emitting progress events with the given event name.
func (a *App) downloadFile(url, dest, progressEvent string) error {
	req, err := http.NewRequestWithContext(a.ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "MavrogUpdater")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 64*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if progressEvent != "" && a.ctx != nil {
				pct := -1.0
				if total > 0 {
					pct = float64(written) / float64(total) * 100
				}
				wruntime.EventsEmit(a.ctx, progressEvent, map[string]interface{}{
					"written": written,
					"total":   total,
					"percent": pct,
				})
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	return nil
}

// extractZipToAddons extracts a zip into the AddOns folder.
// It expects the zip to contain one or more top-level addon folders.
// Returns the list of top-level folder names extracted.
func extractZipToAddons(zipPath, addonsPath string) ([]string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	topLevel := map[string]bool{}
	for _, f := range zr.File {
		clean := filepath.ToSlash(filepath.Clean(f.Name))
		if clean == "." || clean == "" {
			continue
		}
		// Reject any path traversal.
		if strings.Contains(clean, "..") {
			return nil, fmt.Errorf("zip contains illegal path: %s", f.Name)
		}
		parts := strings.SplitN(clean, "/", 2)
		top := parts[0]
		if top == "" {
			continue
		}
		topLevel[top] = true

		dest := filepath.Join(addonsPath, filepath.FromSlash(clean))
		// Ensure destination is inside addonsPath.
		absAddons, _ := filepath.Abs(addonsPath)
		absDest, _ := filepath.Abs(dest)
		if !strings.HasPrefix(absDest, absAddons) {
			return nil, fmt.Errorf("zip entry escapes target: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return nil, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, err
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		out, err := os.Create(dest)
		if err != nil {
			rc.Close()
			return nil, err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return nil, err
		}
		out.Close()
		rc.Close()
	}

	var folders []string
	for k := range topLevel {
		folders = append(folders, k)
	}
	return folders, nil
}

// removeExistingAddon removes an existing addon folder before extraction.
func removeExistingAddon(addonsPath, folder string) error {
	target := filepath.Join(addonsPath, folder)
	if _, err := os.Stat(target); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.RemoveAll(target)
}

// emitLog emits a string log line to the frontend.
func (a *App) emitLog(msg string) {
	if a.ctx == nil {
		return
	}
	wruntime.EventsEmit(a.ctx, "log", msg)
}

// ensure context is captured (avoid unused import if other helpers change)
var _ = context.Background
