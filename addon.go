package main

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// AddonFolderName is the folder name expected inside <WoW>/_retail_/Interface/AddOns
// and inside the release zip.
const AddonFolderName = "MavrogBattlecry"

// detectAddonsFolder tries common Retail WoW install locations.
// Returns "" if none found.
func detectAddonsFolder() string {
	candidates := []string{}
	for _, drive := range []string{"C:", "D:", "E:", "F:"} {
		candidates = append(candidates,
			filepath.Join(drive+`\`, "Program Files (x86)", "World of Warcraft", "_retail_", "Interface", "AddOns"),
			filepath.Join(drive+`\`, "Program Files", "World of Warcraft", "_retail_", "Interface", "AddOns"),
			filepath.Join(drive+`\`, "World of Warcraft", "_retail_", "Interface", "AddOns"),
			filepath.Join(drive+`\`, "Games", "World of Warcraft", "_retail_", "Interface", "AddOns"),
			filepath.Join(drive+`\`, "Battle.net", "World of Warcraft", "_retail_", "Interface", "AddOns"),
		)
	}
	if pf86 := os.Getenv("ProgramFiles(x86)"); pf86 != "" {
		candidates = append(candidates, filepath.Join(pf86, "World of Warcraft", "_retail_", "Interface", "AddOns"))
	}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		candidates = append(candidates, filepath.Join(pf, "World of Warcraft", "_retail_", "Interface", "AddOns"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}

// readInstalledVersion parses the addon's .toc file to extract `## Version:` value.
// Returns "" if not installed.
func readInstalledVersion(addonsPath string) (string, error) {
	if addonsPath == "" {
		return "", errors.New("addons path not configured")
	}
	addonDir := filepath.Join(addonsPath, AddonFolderName)
	st, err := os.Stat(addonDir)
	if err != nil || !st.IsDir() {
		return "", nil // not installed
	}
	// Try common toc names; WoW may use Mainline/-Mainline suffix.
	names := []string{
		AddonFolderName + ".toc",
		AddonFolderName + "_Mainline.toc",
		AddonFolderName + "-Mainline.toc",
	}
	for _, n := range names {
		p := filepath.Join(addonDir, n)
		if v, err := readTocVersion(p); err == nil && v != "" {
			return v, nil
		}
	}
	// Fallback: scan for any .toc.
	entries, err := os.ReadDir(addonDir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".toc") {
			if v, err := readTocVersion(filepath.Join(addonDir, e.Name())); err == nil && v != "" {
				return v, nil
			}
		}
	}
	return "", nil
}

func readTocVersion(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		// strip BOM
		line = strings.TrimPrefix(line, "\ufeff")
		if !strings.HasPrefix(line, "##") {
			continue
		}
		body := strings.TrimSpace(strings.TrimPrefix(line, "##"))
		if i := strings.Index(body, ":"); i > 0 {
			key := strings.TrimSpace(body[:i])
			val := strings.TrimSpace(body[i+1:])
			if strings.EqualFold(key, "Version") {
				return val, nil
			}
		}
	}
	return "", sc.Err()
}
