package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// pickUpdaterAsset chooses an updater binary asset for the current platform.
func pickUpdaterAsset(r *Release) *ReleaseAsset {
	if r == nil {
		return nil
	}
	wantOS := runtime.GOOS // "windows"
	wantExt := ".exe"
	if wantOS != "windows" {
		wantExt = ""
	}
	var fallback *ReleaseAsset
	for i := range r.Assets {
		a := &r.Assets[i]
		lower := strings.ToLower(a.Name)
		if wantExt != "" && strings.HasSuffix(lower, wantExt) {
			return a
		}
		if strings.Contains(lower, wantOS) {
			fallback = a
		}
	}
	return fallback
}

// performSelfUpdate downloads the new updater binary and stages a swap.
// On Windows the running .exe cannot be deleted, but it CAN be renamed.
// Strategy: rename current exe -> .old, place new exe at original path, restart.
// Old file is cleaned up at next startup.
func (a *App) performSelfUpdate(downloadURL string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}

	tmp := exe + ".new"
	old := exe + ".old"

	// Download to .new
	if err := a.downloadFile(downloadURL, tmp, "selfupdate:progress"); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// Remove any leftover .old
	_ = os.Remove(old)

	// Rename current -> .old
	if err := os.Rename(exe, old); err != nil {
		return fmt.Errorf("rename current: %w", err)
	}

	// Move .new -> current path
	if err := os.Rename(tmp, exe); err != nil {
		// Try to roll back
		_ = os.Rename(old, exe)
		return fmt.Errorf("install new: %w", err)
	}

	// Release the single-instance mutex BEFORE spawning the child so the
	// new process can immediately acquire it without retrying.
	if singleInstanceMutex != 0 {
		windows.CloseHandle(singleInstanceMutex)
		singleInstanceMutex = 0
	}

	// Start new exe detached.
	cmd := exec.Command(exe)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008} // DETACHED_PROCESS
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	// Give the new process a moment to start, then hard-exit.
	time.Sleep(200 * time.Millisecond)
	os.Exit(0)
	return nil
}

// cleanupOldExe removes the .old leftover from a previous self-update.
func cleanupOldExe() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	_ = os.Remove(exe + ".old")
}
