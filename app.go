package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// AppVersion is the version of this updater binary.
// Override at build time with: -ldflags "-X main.AppVersion=v1.0.0"
var AppVersion = "v0.1.0"

// App is the main backend struct exposed to the frontend.
type App struct {
	ctx      context.Context
	bgCancel context.CancelFunc
	bgMu     sync.Mutex
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cleanupOldExe()
	a.startBackgroundChecker()
}

// ----- Status / Config -----

// AppStatus is the snapshot returned to the frontend on load.
type AppStatus struct {
	AppVersion       string `json:"appVersion"`
	AddonsPath       string `json:"addonsPath"`
	AddonsAutoFound  bool   `json:"addonsAutoFound"`
	InstalledVersion string `json:"installedVersion"`
	AddonInstalled   bool   `json:"addonInstalled"`
	AutoCheck        bool   `json:"autoCheck"`
	AddonName        string `json:"addonName"`
	AddonRepo        string `json:"addonRepo"`
	ElvUIInstalled   bool   `json:"elvuiInstalled"`
}

// GetStatus is called on startup by the UI.
func (a *App) GetStatus() (AppStatus, error) {
	cfg, _ := loadConfig()
	autoFound := false
	if cfg.AddonsPath == "" {
		if p := detectAddonsFolder(); p != "" {
			cfg.AddonsPath = p
			autoFound = true
			_ = saveConfig(cfg)
		}
	}
	installed, _ := readInstalledVersion(cfg.AddonsPath)
	addonInstalled := installed != ""
	elvuiInstalled := isElvUIInstalled(cfg.AddonsPath)
	return AppStatus{
		AppVersion:       AppVersion,
		AddonsPath:       cfg.AddonsPath,
		AddonsAutoFound:  autoFound,
		InstalledVersion: installed,
		AddonInstalled:   addonInstalled,
		AutoCheck:        cfg.AutoCheck,
		AddonName:        AddonFolderName,
		AddonRepo:        fmt.Sprintf("%s/%s", AddonRepoOwner, AddonRepoName),
		ElvUIInstalled:   elvuiInstalled,
	}, nil
}

// SetAutoCheck persists the auto-check preference.
func (a *App) SetAutoCheck(v bool) error {
	cfg, _ := loadConfig()
	cfg.AutoCheck = v
	return saveConfig(cfg)
}

// PickAddonsFolder opens a directory picker and persists the chosen folder.
func (a *App) PickAddonsFolder() (string, error) {
	if a.ctx == nil {
		return "", errors.New("no context")
	}
	cfg, _ := loadConfig()
	defaultDir := cfg.AddonsPath
	if defaultDir == "" {
		defaultDir = detectAddonsFolder()
	}
	chosen, err := wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title:            "Select your WoW _retail_/Interface/AddOns folder",
		DefaultDirectory: defaultDir,
	})
	if err != nil {
		return "", err
	}
	if chosen == "" {
		return cfg.AddonsPath, nil
	}
	// Soft validation: warn if the folder doesn't look like AddOns.
	lower := strings.ToLower(chosen)
	if !strings.Contains(lower, "addons") {
		// Allow but emit a log warning.
		a.emitLog("Warning: selected folder does not contain 'AddOns' in its path.")
	}
	cfg.AddonsPath = chosen
	if err := saveConfig(cfg); err != nil {
		return chosen, err
	}
	return chosen, nil
}

// ----- Addon update flow -----

// UpdateInfo describes a remote release vs the installed version.
type UpdateInfo struct {
	InstalledVersion string `json:"installedVersion"`
	LatestVersion    string `json:"latestVersion"`
	ReleaseName      string `json:"releaseName"`
	Changelog        string `json:"changelog"`
	HTMLURL          string `json:"htmlUrl"`
	PublishedAt      string `json:"publishedAt"`
	AssetName        string `json:"assetName"`
	AssetURL         string `json:"assetUrl"`
	AssetSize        int64  `json:"assetSize"`
	UpdateAvailable  bool   `json:"updateAvailable"`
	HasAsset         bool   `json:"hasAsset"`
}

// CheckForUpdate queries GitHub for the latest addon release.
func (a *App) CheckForUpdate() (UpdateInfo, error) {
	var info UpdateInfo
	cfg, _ := loadConfig()
	installed, _ := readInstalledVersion(cfg.AddonsPath)
	info.InstalledVersion = installed

	rel, err := fetchLatestRelease(AddonRepoOwner, AddonRepoName)
	if err != nil {
		return info, err
	}
	info.LatestVersion = strings.TrimPrefix(rel.TagName, "v")
	info.ReleaseName = rel.Name
	info.Changelog = rel.Body
	info.HTMLURL = rel.HTMLURL
	if !rel.PublishedAt.IsZero() {
		info.PublishedAt = rel.PublishedAt.Format("2006-01-02 15:04 MST")
	}
	if asset := pickAddonAsset(rel); asset != nil {
		info.AssetName = asset.Name
		info.AssetURL = asset.BrowserDownloadURL
		info.AssetSize = asset.Size
		info.HasAsset = true
	}
	info.UpdateAvailable = info.HasAsset && semverNewer(installed, info.LatestVersion)
	return info, nil
}

func versionsEqual(a, b string) bool {
	na := strings.TrimPrefix(strings.TrimSpace(a), "v")
	nb := strings.TrimPrefix(strings.TrimSpace(b), "v")
	return na != "" && na == nb
}

// semverNewer returns true if b is strictly newer than a.
// Falls back to string equality check if parsing fails.
func semverNewer(current, latest string) bool {
	ca := strings.TrimPrefix(strings.TrimSpace(current), "v")
	cb := strings.TrimPrefix(strings.TrimSpace(latest), "v")
	if ca == "" || cb == "" {
		return false
	}
	if ca == cb {
		return false
	}
	ap := parseSemver(ca)
	bp := parseSemver(cb)
	if ap == nil || bp == nil {
		return false
	}
	if bp[0] != ap[0] {
		return bp[0] > ap[0]
	}
	if bp[1] != ap[1] {
		return bp[1] > ap[1]
	}
	return bp[2] > ap[2]
}

func parseSemver(s string) []int {
	// Strip pre-release suffix (e.g. "1.2.3-beta")
	if i := strings.IndexByte(s, '-'); i > 0 {
		s = s[:i]
	}
	parts := strings.SplitN(s, ".", 3)
	if len(parts) < 2 {
		return nil
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil
			}
			n = n*10 + int(c-'0')
		}
		nums[i] = n
	}
	return nums
}

// ----- ElvUI -----

// CheckElvUIUpdate queries the Tukui API for the latest ElvUI version.
func (a *App) CheckElvUIUpdate() (ElvUIInfo, error) {
	cfg, _ := loadConfig()
	info := ElvUIInfo{
		Installed:        isElvUIInstalled(cfg.AddonsPath),
		InstalledVersion: readElvUIVersion(cfg.AddonsPath),
	}
	if !info.Installed {
		return info, nil
	}
	rel, err := fetchElvUIRelease()
	if err != nil {
		return info, err
	}
	info.LatestVersion = rel.Version
	info.DownloadURL = rel.URL
	info.Changelog = rel.Changelog
	info.WebURL = rel.WebURL
	info.UpdateAvailable = rel.Version != "" && semverNewer(info.InstalledVersion, rel.Version)
	return info, nil
}

// InstallElvUIUpdate downloads and extracts the latest ElvUI zip.
func (a *App) InstallElvUIUpdate() (string, error) {
	cfg, _ := loadConfig()
	if cfg.AddonsPath == "" {
		return "", errors.New("AddOns folder is not configured")
	}
	rel, err := fetchElvUIRelease()
	if err != nil {
		return "", err
	}
	if rel.URL == "" {
		return "", errors.New("no download URL from Tukui API")
	}
	tmpDir, err := os.MkdirTemp("", "mavrog-elvui-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "elvui.zip")
	a.emitLog(fmt.Sprintf("Downloading ElvUI %s...", rel.Version))
	if err := a.downloadFile(rel.URL, zipPath, "install:progress"); err != nil {
		return "", err
	}
	a.emitLog("Removing previous ElvUI install...")
	for _, folder := range []string{"ElvUI", "ElvUI_Options"} {
		_ = removeExistingAddon(cfg.AddonsPath, folder)
	}
	a.emitLog("Extracting ElvUI...")
	folders, err := extractZipToAddons(zipPath, cfg.AddonsPath)
	if err != nil {
		return "", err
	}
	a.emitLog(fmt.Sprintf("ElvUI installed: %s", strings.Join(folders, ", ")))
	wruntime.EventsEmit(a.ctx, "elvui:installed", rel.Version)
	return rel.Version, nil
}

// ----- Install prompt -----

// ConfirmInstallAddon is called when the user confirms they want to install MavrogBattlecry.
func (a *App) ConfirmInstallAddon() (string, error) {
	return a.InstallUpdate()
}

// ----- Background checker -----

func (a *App) startBackgroundChecker() {
	a.bgMu.Lock()
	defer a.bgMu.Unlock()
	if a.bgCancel != nil {
		a.bgCancel()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.bgCancel = cancel
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.runBackgroundCheck()
			}
		}
	}()
}

func (a *App) runBackgroundCheck() {
	cfg, _ := loadConfig()
	installed, _ := readInstalledVersion(cfg.AddonsPath)
	rel, err := fetchLatestRelease(AddonRepoOwner, AddonRepoName)
	if err != nil {
		return
	}
	latest := strings.TrimPrefix(rel.TagName, "v")
	if semverNewer(installed, latest) {
		sendToast(
			"MavrogBattlecry Update Available",
			fmt.Sprintf("Version %s is ready to install.", latest),
			"mavrog-updater://open",
		)
		if a.ctx != nil {
			wruntime.EventsEmit(a.ctx, "bg:update", map[string]string{
				"latest": latest,
			})
		}
	}
}

// InstallUpdate downloads the latest addon release zip and extracts it into AddOns.
func (a *App) InstallUpdate() (string, error) {
	cfg, _ := loadConfig()
	if cfg.AddonsPath == "" {
		return "", errors.New("AddOns folder is not configured")
	}
	if st, err := os.Stat(cfg.AddonsPath); err != nil || !st.IsDir() {
		return "", fmt.Errorf("AddOns folder is invalid: %s", cfg.AddonsPath)
	}

	a.emitLog("Checking latest release...")
	rel, err := fetchLatestRelease(AddonRepoOwner, AddonRepoName)
	if err != nil {
		return "", err
	}
	asset := pickAddonAsset(rel)
	if asset == nil {
		return "", errors.New("no .zip asset found on the latest release")
	}

	tmpDir, err := os.MkdirTemp("", "mavrog-updater-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, asset.Name)
	a.emitLog(fmt.Sprintf("Downloading %s ...", asset.Name))
	if err := a.downloadFile(asset.BrowserDownloadURL, zipPath, "install:progress"); err != nil {
		return "", err
	}

	a.emitLog("Removing previous install...")
	if err := removeExistingAddon(cfg.AddonsPath, AddonFolderName); err != nil {
		return "", fmt.Errorf("cleanup: %w", err)
	}

	a.emitLog("Extracting...")
	folders, err := extractZipToAddons(zipPath, cfg.AddonsPath)
	if err != nil {
		return "", err
	}

	cfg.LastVersion = strings.TrimPrefix(rel.TagName, "v")
	_ = saveConfig(cfg)

	a.emitLog(fmt.Sprintf("Installed: %s", strings.Join(folders, ", ")))
	wruntime.EventsEmit(a.ctx, "addon:installed", cfg.LastVersion)
	return cfg.LastVersion, nil
}

// ----- Self update -----

// SelfUpdateInfo represents an update for the updater binary itself.
type SelfUpdateInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	HasAsset        bool   `json:"hasAsset"`
	AssetURL        string `json:"assetUrl"`
	AssetName       string `json:"assetName"`
	HTMLURL         string `json:"htmlUrl"`
	Changelog       string `json:"changelog"`
}

// CheckSelfUpdate queries the updater's own release repo for a newer binary.
func (a *App) CheckSelfUpdate() (SelfUpdateInfo, error) {
	info := SelfUpdateInfo{CurrentVersion: AppVersion}
	rel, err := fetchLatestRelease(UpdaterRepoOwner, UpdaterRepoName)
	if err != nil {
		return info, err
	}
	info.LatestVersion = rel.TagName
	info.HTMLURL = rel.HTMLURL
	info.Changelog = rel.Body
	if asset := pickUpdaterAsset(rel); asset != nil {
		info.AssetURL = asset.BrowserDownloadURL
		info.AssetName = asset.Name
		info.HasAsset = true
	}
	info.UpdateAvailable = info.HasAsset && semverNewer(AppVersion, rel.TagName)
	return info, nil
}

// ApplySelfUpdate downloads the updater's new binary and restarts.
func (a *App) ApplySelfUpdate() error {
	su, err := a.CheckSelfUpdate()
	if err != nil {
		return err
	}
	if !su.HasAsset {
		return errors.New("no compatible binary asset on the latest updater release")
	}
	a.emitLog("Downloading updater " + su.LatestVersion + "...")
	return a.performSelfUpdate(su.AssetURL)
}

// OpenURL opens a URL in the user's default browser.
func (a *App) OpenURL(url string) {
	if a.ctx == nil {
		return
	}
	wruntime.BrowserOpenURL(a.ctx, url)
}

// OpenAddonsFolder opens the configured AddOns folder in the OS file manager.
func (a *App) OpenAddonsFolder() error {
	cfg, _ := loadConfig()
	if cfg.AddonsPath == "" {
		return errors.New("addons path not set")
	}
	if a.ctx != nil {
		wruntime.BrowserOpenURL(a.ctx, "file:///"+filepath.ToSlash(cfg.AddonsPath))
	}
	return nil
}
